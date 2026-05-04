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

type levelEnemySpawn struct {
	entity       levels.Entity
	gameEntityID string
}

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

// ResetCurrentLevelEnemies rebuilds authored enemy entities for the current
// level from level data without reloading the world.
func ResetCurrentLevelEnemies(world *ecs.World, levelName string, lvl *levels.Level, stateMap *component.LevelEntityStateMap) (int, error) {
	if world == nil || lvl == nil {
		return 0, nil
	}

	enemySpawns := collectLevelEnemySpawns(lvl)
	if len(enemySpawns) == 0 {
		return 0, nil
	}

	enemyIDs := make(map[string]struct{}, len(enemySpawns))
	for _, spawn := range enemySpawns {
		enemyIDs[spawn.gameEntityID] = struct{}{}
	}

	clearPersistedEnemyStates(levelName, enemyIDs, stateMap)
	destroyAliveLevelEnemies(world, enemyIDs)

	respawned := 0
	for _, spawn := range enemySpawns {
		if err := spawnGenericLevelEntity(world, lvl, spawn.entity, spawn.gameEntityID); err != nil {
			return respawned, err
		}
		respawned++
	}

	return respawned, nil
}

func collectLevelEnemySpawns(lvl *levels.Level) []levelEnemySpawn {
	if lvl == nil || len(lvl.Entities) == 0 {
		return nil
	}

	usedGameEntityIDs := map[string]bool{}
	enemySpawns := make([]levelEnemySpawn, 0)
	for i, ent := range lvl.Entities {
		gameEntityID := levelEntityGameID(i, ent.ID, usedGameEntityIDs)
		if !levelEntityIsEnemy(ent) {
			continue
		}
		enemySpawns = append(enemySpawns, levelEnemySpawn{entity: ent, gameEntityID: gameEntityID})
	}

	return enemySpawns
}

func clearPersistedEnemyStates(levelName string, enemyIDs map[string]struct{}, stateMap *component.LevelEntityStateMap) {
	if levelName == "" || len(enemyIDs) == 0 || stateMap == nil || len(stateMap.States) == 0 {
		return
	}

	for gameEntityID := range enemyIDs {
		delete(stateMap.States, levelName+"#"+gameEntityID)
	}
}

func destroyAliveLevelEnemies(world *ecs.World, enemyIDs map[string]struct{}) {
	if world == nil || len(enemyIDs) == 0 {
		return
	}

	toDestroy := make([]ecs.Entity, 0)
	ecs.ForEach(world, component.GameEntityIDComponent.Kind(), func(e ecs.Entity, id *component.GameEntityID) {
		if id == nil || !ecs.IsAlive(world, e) {
			return
		}
		if _, ok := enemyIDs[id.Value]; !ok {
			return
		}
		toDestroy = append(toDestroy, e)
	})

	for _, e := range toDestroy {
		if ecs.IsAlive(world, e) {
			ecs.DestroyEntity(world, e)
		}
	}
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
