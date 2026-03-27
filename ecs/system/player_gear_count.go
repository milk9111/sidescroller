package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const playerGearCountPersistentID = "player_gears"

func ensurePlayerGearCount(w *ecs.World) *component.PlayerGearCount {
	if w == nil {
		return nil
	}

	if ent, ok := ecs.First(w, component.PlayerGearCountComponent.Kind()); ok {
		if gears, ok := ecs.Get(w, ent, component.PlayerGearCountComponent.Kind()); ok && gears != nil {
			return gears
		}
	}

	ent := ecs.CreateEntity(w)
	gears := &component.PlayerGearCount{}
	_ = ecs.Add(w, ent, component.PersistentComponent.Kind(), &component.Persistent{
		ID:                playerGearCountPersistentID,
		KeepOnLevelChange: true,
		KeepOnReload:      false,
	})
	_ = ecs.Add(w, ent, component.PlayerGearCountComponent.Kind(), gears)
	return gears
}

func currentPlayerGearCount(w *ecs.World) int {
	if w == nil {
		return 0
	}

	if ent, ok := ecs.First(w, component.PlayerGearCountComponent.Kind()); ok {
		if gears, ok := ecs.Get(w, ent, component.PlayerGearCountComponent.Kind()); ok && gears != nil {
			return gears.Count
		}
	}

	return 0
}
