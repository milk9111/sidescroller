package systems

import (
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// DamageSystem applies knockback or other effects from combat events.
type DamageSystem struct {
	Events *[]component.CombatEvent
}

// NewDamageSystem creates a DamageSystem.
func NewDamageSystem(events *[]component.CombatEvent) *DamageSystem {
	return &DamageSystem{Events: events}
}

// Update applies knockback to targets from damage events.
func (s *DamageSystem) Update(w *ecs.World) {
	if w == nil || s.Events == nil || len(*s.Events) == 0 {
		return
	}
	vels := w.Velocities()
	bodies := w.PhysicsBodies()
	if vels == nil && bodies == nil {
		return
	}

	for _, evt := range *s.Events {
		if evt.Type != component.EventDamageApplied {
			continue
		}
		id := evt.TargetID
		if id <= 0 {
			continue
		}
		if bodies != nil {
			if bv := bodies.Get(id); bv != nil {
				if body, ok := bv.(*components.PhysicsBody); ok && body != nil && body.Body != nil {
					v := body.Body.Velocity()
					body.Body.SetVelocity(v.X+float64(evt.KnockbackX), v.Y+float64(evt.KnockbackY))
					continue
				}
			}
		}
		if vels != nil {
			if vv := vels.Get(id); vv != nil {
				if vel, ok := vv.(*components.Velocity); ok && vel != nil {
					vel.VX += evt.KnockbackX
					vel.VY += evt.KnockbackY
				}
			}
		}
	}
}
