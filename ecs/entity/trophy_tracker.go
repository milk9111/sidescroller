package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const trophyTrackerPersistentID = "trophy_tracker"

func NewTrophyTracker(w *ecs.World) (ecs.Entity, error) {
	ent := ecs.CreateEntity(w)
	if err := ecs.Add(w, ent, component.PersistentComponent.Kind(), &component.Persistent{
		ID:                trophyTrackerPersistentID,
		KeepOnLevelChange: true,
		KeepOnReload:      false,
	}); err != nil {
		return 0, fmt.Errorf("trophy tracker: add persistent component: %w", err)
	}

	if err := ecs.Add(w, ent, component.TrophyTrackerComponent.Kind(), &component.TrophyTracker{}); err != nil {
		return 0, fmt.Errorf("trophy tracker: add tracker component: %w", err)
	}

	return ent, nil
}
