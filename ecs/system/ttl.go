package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

// TTLSystem decrements frame-based TTL components and destroys entities when
// the TTL reaches zero.
type TTLSystem struct{}

func NewTTLSystem() *TTLSystem {
	return &TTLSystem{}
}

func (s *TTLSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach(w, component.TTLComponent.Kind(), func(e ecs.Entity, ttl *component.TTL) {
		if ttl == nil {
			return
		}

		if ttl.Frames > 0 {
			ttl.Frames--
			if ttl.Frames > 0 {
				_ = ecs.Add(w, e, component.TTLComponent.Kind(), ttl)
				return
			}
		}

		// TTL expired: destroy the entity
		ecs.DestroyEntity(w, e)
	})
}
