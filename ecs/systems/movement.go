package systems

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// MovementSystem applies velocity and acceleration to transforms or physics bodies.
type MovementSystem struct{}

// NewMovementSystem creates a MovementSystem.
func NewMovementSystem() *MovementSystem {
	return &MovementSystem{}
}

// Update applies acceleration, gravity, and movement.
func (s *MovementSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	vels := w.Velocities()
	trs := w.Transforms()
	accs := w.Accelerations()
	gravs := w.Gravities()
	bodies := w.PhysicsBodies()

	if vels == nil || trs == nil {
		return
	}

	for _, id := range vels.Entities() {
		v := vels.Get(id)
		vel, ok := v.(*components.Velocity)
		if !ok || vel == nil {
			continue
		}
		tv := trs.Get(id)
		tr, ok := tv.(*components.Transform)
		if !ok || tr == nil {
			continue
		}

		if accs != nil {
			if av := accs.Get(id); av != nil {
				if acc, ok := av.(*components.Acceleration); ok && acc != nil {
					vel.VX += acc.AX
					vel.VY += acc.AY
				}
			}
		}
		if gravs != nil {
			if gv := gravs.Get(id); gv != nil {
				if grav, ok := gv.(*components.Gravity); ok && grav != nil && grav.Enabled {
					vel.VY += grav.Value
				}
			}
		}

		if bodies != nil {
			if bv := bodies.Get(id); bv != nil {
				if body, ok := bv.(*components.PhysicsBody); ok && body != nil && body.Body != nil {
					body.Body.SetVelocity(float64(vel.VX), float64(vel.VY))
					continue
				}
			}
		}

		tr.X += vel.VX
		tr.Y += vel.VY
	}
}
