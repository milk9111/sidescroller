package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func ensurePlayerLevelEntityStateMap(w *ecs.World) *component.LevelEntityStateMap {
	if w == nil {
		return nil
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return nil
	}

	stateMap, ok := ecs.Get(w, player, component.LevelEntityStateMapComponent.Kind())
	if ok && stateMap != nil {
		if stateMap.States == nil {
			stateMap.States = make(map[string]component.PersistedLevelEntityState)
		}
		return stateMap
	}

	stateMap = &component.LevelEntityStateMap{States: make(map[string]component.PersistedLevelEntityState)}
	_ = ecs.Add(w, player, component.LevelEntityStateMapComponent.Kind(), stateMap)
	return stateMap
}

func currentLevelName(w *ecs.World) string {
	if w == nil {
		return ""
	}

	ent, ok := ecs.First(w, component.LevelRuntimeComponent.Kind())
	if !ok {
		return ""
	}

	runtimeComp, ok := ecs.Get(w, ent, component.LevelRuntimeComponent.Kind())
	if !ok || runtimeComp == nil {
		return ""
	}

	return runtimeComp.Name
}

func levelEntityStateKey(levelName, gameEntityID string) string {
	if levelName == "" || gameEntityID == "" {
		return ""
	}
	return levelName + "#" + gameEntityID
}

func levelEntityStateKeyForEntity(w *ecs.World, e ecs.Entity) string {
	if w == nil {
		return ""
	}

	id, ok := ecs.Get(w, e, component.GameEntityIDComponent.Kind())
	if !ok || id == nil || id.Value == "" {
		return ""
	}

	return levelEntityStateKey(currentLevelName(w), id.Value)
}

func recordLevelEntityState(w *ecs.World, e ecs.Entity, state component.PersistedLevelEntityState) bool {
	if w == nil || state == "" {
		return false
	}

	key := levelEntityStateKeyForEntity(w, e)
	if key == "" {
		return false
	}

	stateMap := ensurePlayerLevelEntityStateMap(w)
	if stateMap == nil {
		return false
	}

	stateMap.States[key] = state
	return true
}

func persistedLevelEntityState(w *ecs.World, e ecs.Entity) (component.PersistedLevelEntityState, bool) {
	if w == nil {
		return "", false
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return "", false
	}

	stateMap, ok := ecs.Get(w, player, component.LevelEntityStateMapComponent.Kind())
	if !ok || stateMap == nil || len(stateMap.States) == 0 {
		return "", false
	}

	state, ok := stateMap.States[levelEntityStateKeyForEntity(w, e)]
	return state, ok
}

func applyPersistedLevelEntityStates(w *ecs.World) {
	if w == nil {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	stateMap, ok := ecs.Get(w, player, component.LevelEntityStateMapComponent.Kind())
	if !ok || stateMap == nil || len(stateMap.States) == 0 {
		return
	}

	toDestroy := make([]ecs.Entity, 0)
	for _, e := range ecs.Entities(w) {
		state, ok := persistedLevelEntityState(w, e)
		if !ok {
			continue
		}

		switch state {
		case component.PersistedLevelEntityStateDefeated:
			if ecs.Has(w, e, component.AITagComponent.Kind()) {
				toDestroy = append(toDestroy, e)
			}
		case component.PersistedLevelEntityStateCollected:
			if ecs.Has(w, e, component.PickupComponent.Kind()) {
				toDestroy = append(toDestroy, e)
			}
		case component.PersistedLevelEntityStateUsed:
			if trigger, ok := ecs.Get(w, e, component.TriggerComponent.Kind()); ok && trigger != nil {
				trigger.Disabled = true
			}
		}
	}

	for _, e := range toDestroy {
		if ecs.IsAlive(w, e) {
			ecs.DestroyEntity(w, e)
		}
	}
}
