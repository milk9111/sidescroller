package system

import (
	"math"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const damageKnockbackImpulse = 4.0
const damageKnockbackMaxDeltaV = 28.0

const strongDamageKnockbackImpulse = 32.0
const strongDamageKnockbackMaxDeltaV = 56.0

type DamageKnockbackSystem struct{}

func NewDamageKnockbackSystem() *DamageKnockbackSystem { return &DamageKnockbackSystem{} }

// Update processes all pending DamageKnockback requests and applies the
// appropriate physics impulse, then removes the request component.
func (s *DamageKnockbackSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	// Process normal and strong knockback requests attached to entities.
	// Only entities that are explicitly Knockbackable will be affected.
	ecs.ForEach4(w, component.DamageKnockbackRequestComponent.Kind(), component.KnockbackableComponent.Kind(), component.TransformComponent.Kind(), component.PhysicsBodyComponent.Kind(), func(e ecs.Entity, req *component.DamageKnockback, _ *component.Knockbackable, t *component.Transform, body *component.PhysicsBody) {
		if req == nil || t == nil || body == nil || body.Body == nil || body.Static {
			_ = ecs.Remove(w, e, component.DamageKnockbackRequestComponent.Kind())
			return
		}

		// If the target has a health component and is out of health, skip.
		if h, hok := ecs.Get(w, e, component.HealthComponent.Kind()); hok && h != nil && h.Current == 0 {
			_ = ecs.Remove(w, e, component.DamageKnockbackRequestComponent.Kind())
			return
		}

		centerX := t.X + body.OffsetX
		centerY := t.Y + body.OffsetY
		if body.AlignTopLeft {
			centerX += body.Width / 2
			centerY += body.Height / 2
		}

		dx := centerX - req.SourceX
		dy := centerY - req.SourceY
		length := math.Hypot(dx, dy)
		if length <= 1e-6 {
			dx = 0
			dy = -1
			length = 1
		}

		nx := dx / length
		ny := dy / length

		if req.Strong {
			// Bias strong knockback for player hit by AI to be more horizontal.
			if ecs.Has(w, e, component.PlayerTagComponent.Kind()) && req.SourceEntity != 0 {
				if ecs.Has(w, ecs.Entity(req.SourceEntity), component.AITagComponent.Kind()) {
					thr := 8.0
					if body.Width > 0 {
						wthr := body.Width * 0.25
						if wthr > thr {
							thr = wthr
						}
					}
					if math.Abs(dx) < thr {
						if req.SourceX > centerX {
							nx = -1.5
						} else {
							nx = 1.5
						}
						ny = math.Copysign(0.25, dy)
					} else {
						if ny > 0.4 {
							ny = 0.4
						} else if ny < -0.4 {
							ny = -0.4
						}
					}
					rlen := math.Hypot(nx, ny)
					if rlen > 1e-6 {
						nx /= rlen
						ny /= rlen
					}
				}
			}

			body.Body.ApplyImpulseAtWorldPoint(
				cp.Vector{X: nx * strongDamageKnockbackImpulse, Y: ny * strongDamageKnockbackImpulse},
				body.Body.Position(),
			)

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
		} else {
			// Normal knockback
			body.Body.ApplyImpulseAtWorldPoint(
				cp.Vector{X: nx * damageKnockbackImpulse, Y: ny * damageKnockbackImpulse},
				body.Body.Position(),
			)

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

		// Remove processed request
		_ = ecs.Remove(w, e, component.DamageKnockbackRequestComponent.Kind())
	})
}
