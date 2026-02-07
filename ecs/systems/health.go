package systems

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// HealthSystem advances i-frame timers on health components.
type HealthSystem struct{}

// NewHealthSystem creates a HealthSystem.
func NewHealthSystem() *HealthSystem {
	return &HealthSystem{}
}

// Update ticks health components.
func (s *HealthSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	set := w.Healths()
	if set == nil {
		return
	}
	for _, id := range set.Entities() {
		if hv := set.Get(id); hv != nil {
			if h, ok := hv.(*components.Health); ok && h != nil {
				h.Tick()
			}
		}
	}
}
