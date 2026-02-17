package system

import (
	"math"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const damageKnockbackImpulse = 8.0
const damageKnockbackMaxDeltaV = 28.0

const strongDamageKnockbackImpulse = 32.0
const strongDamageKnockbackMaxDeltaV = 56.0

func applyDamageKnockback(w *ecs.World, target ecs.Entity, sourceX, sourceY float64) {
	if w == nil {
		return
	}
	body, ok := ecs.Get(w, target, component.PhysicsBodyComponent.Kind())
	if !ok || body == nil || body.Body == nil || body.Static {
		return
	}
	t, ok := ecs.Get(w, target, component.TransformComponent.Kind())
	if !ok || t == nil {
		return
	}

	centerX := t.X + body.OffsetX
	centerY := t.Y + body.OffsetY
	if body.AlignTopLeft {
		centerX += body.Width / 2
		centerY += body.Height / 2
	}

	dx := centerX - sourceX
	dy := centerY - sourceY
	length := math.Hypot(dx, dy)
	if length <= 1e-6 {
		dx = 0
		dy = -1
		length = 1
	}

	nx := dx / length
	ny := dy / length

	// Apply a short impulse rather than directly writing a large velocity.
	// This keeps damage response snappy without looking like teleportation.
	body.Body.ApplyImpulseAtWorldPoint(
		cp.Vector{X: nx * damageKnockbackImpulse, Y: ny * damageKnockbackImpulse},
		body.Body.Position(),
	)

	// Safety clamp: cap total velocity delta contributed by knockback in a
	// single hit so stacked overlaps don't launch entities too hard.
	v := body.Body.Velocity()
	vDot := v.X*nx + v.Y*ny
	if vDot > damageKnockbackMaxDeltaV {
		tx := v.X - nx*vDot
		ty := v.Y - ny*vDot
		body.Body.SetVelocityVector(cp.Vector{
			X: tx + nx*damageKnockbackMaxDeltaV,
			Y: ty + ny*damageKnockbackMaxDeltaV,
		})
	}
}

// applyStrongDamageKnback applies a stronger impulse for hazards and other
// high-impact collisions (e.g., player running into enemy).
func applyStrongDamageKnockback(w *ecs.World, target ecs.Entity, sourceX, sourceY float64, sourceEntity ecs.Entity) {
	if w == nil {
		return
	}
	body, ok := ecs.Get(w, target, component.PhysicsBodyComponent.Kind())
	if !ok || body == nil || body.Body == nil || body.Static {
		return
	}
	t, ok := ecs.Get(w, target, component.TransformComponent.Kind())
	if !ok || t == nil {
		return
	}

	centerX := t.X + body.OffsetX
	centerY := t.Y + body.OffsetY
	if body.AlignTopLeft {
		centerX += body.Width / 2
		centerY += body.Height / 2
	}

	dx := centerX - sourceX
	dy := centerY - sourceY
	length := math.Hypot(dx, dy)
	if length <= 1e-6 {
		dx = 0
		dy = -1
		length = 1
	}

	nx := dx / length
	ny := dy / length

	// If the target is the player and the source is an AI, bias the
	// knockback to be more horizontal so the player is knocked left/right
	// instead of strongly up/down. If the attacker is roughly vertically
	// aligned with the player (small dx), force a horizontal push away
	// from the attacker; otherwise clamp the vertical component to keep
	// the effect feeling like a side-bounce.
	if ecs.Has(w, target, component.PlayerTagComponent.Kind()) && sourceEntity != 0 {
		if ecs.Has(w, sourceEntity, component.AITagComponent.Kind()) {
			// decide threshold based on body width (fallback to 8px)
			thr := 8.0
			if body.Width > 0 {
				wthr := body.Width * 0.25
				if wthr > thr {
					thr = wthr
				}
			}
			if math.Abs(dx) < thr {
				// attacker is mostly above/below: strongly push the player horizontally
				if sourceX > centerX {
					nx = -1.5
				} else {
					nx = 1.5
				}
				// keep a small vertical component away from the source
				ny = math.Copysign(0.25, dy)
			} else {
				// attacker is to the side: limit vertical influence so knock
				// is strongly horizontal but still away.
				if ny > 0.4 {
					ny = 0.4
				} else if ny < -0.4 {
					ny = -0.4
				}
			}
			// renormalize
			rlen := math.Hypot(nx, ny)
			if rlen > 1e-6 {
				nx /= rlen
				ny /= rlen
			}
		}
	}

	// Apply the strong impulse using the computed normal.
	body.Body.ApplyImpulseAtWorldPoint(
		cp.Vector{X: nx * strongDamageKnockbackImpulse, Y: ny * strongDamageKnockbackImpulse},
		body.Body.Position(),
	)

	// Safety clamp: cap total velocity delta contributed by knockback in a
	// single hit so stacked overlaps don't launch entities too hard.
	v := body.Body.Velocity()
	vDot := v.X*nx + v.Y*ny
	if vDot > strongDamageKnockbackMaxDeltaV {
		tx := v.X - nx*vDot
		ty := v.Y - ny*vDot
		body.Body.SetVelocityVector(cp.Vector{
			X: tx + nx*strongDamageKnockbackMaxDeltaV,
			Y: ty + ny*strongDamageKnockbackMaxDeltaV,
		})
	}
}
