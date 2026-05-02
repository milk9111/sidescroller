package entity

import (
	"fmt"
	"math"
	"strings"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/prefabs"
)

// RespawnDefeatedLevelEnemies rebuilds defeated enemy entities for the current
// level without reloading the world, avoiding camera/render artifacts.
func RespawnDefeatedLevelEnemies(world *ecs.World, levelName string, lvl *levels.Level, stateMap *component.LevelEntityStateMap) (int, error) {
	if world == nil || lvl == nil || stateMap == nil || len(stateMap.States) == 0 {
		return 0, nil
	}

	usedGameEntityIDs := map[string]bool{}
	respawned := 0
	for i, ent := range lvl.Entities {
		gameEntityID := levelEntityGameID(i, ent.ID, usedGameEntityIDs)
		key := levelName + "#" + gameEntityID
		state, ok := stateMap.States[key]
		if !ok || state != component.PersistedLevelEntityStateDefeated {
			continue
		}

		if !levelEntityIsEnemy(ent) {
			continue
		}

		delete(stateMap.States, key)
		if hasAliveEntityWithGameID(world, gameEntityID) {
			continue
		}

		if err := spawnGenericLevelEntity(world, lvl, ent, gameEntityID); err != nil {
			return respawned, err
		}
		respawned++
	}

	return respawned, nil
}

func levelEntityGameID(index int, rawID string, used map[string]bool) string {
	gameEntityID := strings.TrimSpace(rawID)
	if gameEntityID == "" {
		gameEntityID = fmt.Sprintf("e%d", index+1)
	}
	for used[gameEntityID] {
		gameEntityID = fmt.Sprintf("%s_%d", gameEntityID, index+1)
	}
	used[gameEntityID] = true
	return gameEntityID
}

func levelEntityIsEnemy(ent levels.Entity) bool {
	prefabPath := prefabPathForLevelEntity(strings.ToLower(ent.Type), ent.Props)
	if strings.TrimSpace(prefabPath) == "" {
		return false
	}
	spec, err := prefabs.LoadEntityBuildSpecWithOverrides(prefabPath, componentOverridesFromLevelProps(ent.Props))
	if err != nil {
		return false
	}
	if len(spec.Components) == 0 {
		return false
	}
	if _, ok := spec.Components["ai"]; ok {
		return true
	}
	if _, ok := spec.Components["ai_tag"]; ok {
		return true
	}
	return false
}

func hasAliveEntityWithGameID(world *ecs.World, gameEntityID string) bool {
	if world == nil || strings.TrimSpace(gameEntityID) == "" {
		return false
	}
	found := false
	ecs.ForEach(world, component.GameEntityIDComponent.Kind(), func(e ecs.Entity, id *component.GameEntityID) {
		if found || id == nil || id.Value != gameEntityID || !ecs.IsAlive(world, e) {
			return
		}
		found = true
	})
	return found
}

func spawnGenericLevelEntity(world *ecs.World, lvl *levels.Level, ent levels.Entity, gameEntityID string) error {
	entityType := strings.ToLower(ent.Type)
	prefabPath := prefabPathForLevelEntity(entityType, ent.Props)
	if prefabPath == "" {
		return nil
	}

	componentOverrides := componentOverridesFromLevelProps(ent.Props)
	layerIndex := levelEntityLayerIndex(ent.Props)
	if !levelLayerActive(lvl, layerIndex) {
		return nil
	}

	e, err := BuildEntityWithOverrides(world, prefabPath, componentOverrides)
	if err != nil {
		return err
	}

	rotation := 0.0
	if entityType == "spike" {
		rotation = toFloat64(ent.Props["rotation"]) * math.Pi / 180.0
	}

	x := float64(ent.X)
	y := float64(ent.Y)
	if entityType == "spike" {
		if s, ok := ecs.Get(world, e, component.SpriteComponent.Kind()); ok && s != nil {
			x += s.OriginX
			y += s.OriginY
		}
	}

	if err := SetEntityTransform(world, e, x, y, rotation); err != nil {
		return err
	}
	if ecs.Has(world, e, component.AreaBoundsComponent.Kind()) {
		if err := applyGenericAreaEntityPlacement(world, e, ent.Props, true); err != nil {
			return err
		}
	}
	if entityType == "spike" {
		_ = ecs.Remove(world, e, component.HazardComponent.Kind())
	}
	if err := ecs.Add(world, e, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: gameEntityID}); err != nil {
		return err
	}
	if err := ecs.Add(world, e, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: layerIndex}); err != nil {
		return err
	}
	return nil
}
