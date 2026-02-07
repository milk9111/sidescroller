package systems

import (
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// HurtboxSyncSystem keeps hurtboxes aligned with transforms and colliders.
type HurtboxSyncSystem struct{}

// NewHurtboxSyncSystem creates a HurtboxSyncSystem.
func NewHurtboxSyncSystem() *HurtboxSyncSystem {
	return &HurtboxSyncSystem{}
}

// Update refreshes hurtbox rects for entities with transforms and colliders.
func (s *HurtboxSyncSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	set := w.Hurtboxes()
	if set == nil {
		return
	}
	trs := w.Transforms()
	cols := w.Colliders()
	if trs == nil || cols == nil {
		return
	}

	for _, id := range set.Entities() {
		hv := set.Get(id)
		hurt, ok := hv.(*components.HurtboxSet)
		if !ok || hurt == nil || len(hurt.Boxes) == 0 {
			continue
		}
		trv := trs.Get(id)
		tr, ok := trv.(*components.Transform)
		if !ok || tr == nil {
			continue
		}
		cv := cols.Get(id)
		col, ok := cv.(*components.Collider)
		if !ok || col == nil {
			continue
		}

		for i := range hurt.Boxes {
			h := hurt.Boxes[i]
			h.Rect = common.Rect{X: tr.X, Y: tr.Y, Width: col.Width, Height: col.Height}
			h.OwnerID = id
			h.Enabled = true
			if h.Faction == component.FactionNeutral {
				h.Faction = hurt.Faction
			}
			hurt.Boxes[i] = h
		}
	}
}
