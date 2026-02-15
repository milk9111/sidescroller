package system

import (
	"math"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const damageKnockbackImpulse = 14.0
const damageKnockbackMaxDeltaV = 28.0

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
