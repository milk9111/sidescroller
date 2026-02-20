package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

// CooldownSystem decrements frame-based cooldowns and notifies AI FSMs
// when a cooldown finishes by adding an AIStateInterruptComponent event
// named "cooldown_finished".
type CooldownSystem struct{}

func NewCooldownSystem() *CooldownSystem {
	return &CooldownSystem{}
}

func (s *CooldownSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach(w, component.CooldownComponent.Kind(), func(e ecs.Entity, cd *component.Cooldown) {
		if cd == nil {
			return
		}
		if cd.Frames > 0 {
			cd.Frames--
			// write back updated component
			_ = ecs.Add(w, e, component.CooldownComponent.Kind(), cd)
			return
		}

		// cooldown finished: remove component and enqueue AI event if applicable
		_ = ecs.Remove(w, e, component.CooldownComponent.Kind())

		// If this entity has an AIStateComponent, signal the FSM with an
		// interrupt that will be consumed by AISystem on its next tick.
		if _, ok := ecs.Get(w, e, component.AIStateComponent.Kind()); ok {
			_ = ecs.Add(w, e, component.AIStateInterruptComponent.Kind(), &component.AIStateInterrupt{Event: "cooldown_finished"})
		}
	})
}
