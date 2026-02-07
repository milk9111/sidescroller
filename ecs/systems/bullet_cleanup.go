package systems

import (
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
)

// BulletCleanupSystem removes bullets after they apply damage.
type BulletCleanupSystem struct {
	Events *[]component.CombatEvent
}

// NewBulletCleanupSystem creates a BulletCleanupSystem.
func NewBulletCleanupSystem(events *[]component.CombatEvent) *BulletCleanupSystem {
	return &BulletCleanupSystem{Events: events}
}

// Update despawns bullets that hit something.
func (s *BulletCleanupSystem) Update(w *ecs.World) {
	if w == nil || s == nil || s.Events == nil || len(*s.Events) == 0 {
		return
	}
	bullets := w.Bullets()
	if bullets == nil {
		return
	}
	for _, evt := range *s.Events {
		if evt.Type != component.EventDamageApplied {
			continue
		}
		if bullets.Has(evt.AttackerID) {
			despawnBullet(w, evt.AttackerID)
		}
	}
}
