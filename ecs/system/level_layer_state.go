package system

import (
	"fmt"
	"strings"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/levels"
)

func ensurePlayerLevelLayerStateMap(w *ecs.World) *component.LevelLayerStateMap {
	if w == nil {
		return nil
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return nil
	}

	stateMap, ok := ecs.Get(w, player, component.LevelLayerStateMapComponent.Kind())
	if ok && stateMap != nil {
		if stateMap.States == nil {
			stateMap.States = make(map[string]bool)
		}
		return stateMap
	}

	stateMap = &component.LevelLayerStateMap{States: make(map[string]bool)}
	_ = ecs.Add(w, player, component.LevelLayerStateMapComponent.Kind(), stateMap)
	return stateMap
}

func levelLayerStateKey(levelName, layerName string) string {
	levelName = strings.TrimSpace(levelName)
	layerName = strings.TrimSpace(layerName)
	if levelName == "" || layerName == "" {
		return ""
	}
	return levelName + "#" + layerName
}

func RecordLevelLayerState(w *ecs.World, layerName string, active bool) bool {
	if w == nil {
		return false
	}

	key := levelLayerStateKey(currentLevelName(w), layerName)
	if key == "" {
		return false
	}

	stateMap := ensurePlayerLevelLayerStateMap(w)
	if stateMap == nil {
		return false
	}

	stateMap.States[key] = active
	return true
}

func applyPersistedLevelLayerStates(w *ecs.World) error {
	if w == nil {
		return nil
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return nil
	}

	stateMap, ok := ecs.Get(w, player, component.LevelLayerStateMapComponent.Kind())
	if !ok || stateMap == nil || len(stateMap.States) == 0 {
		return nil
	}

	runtimeComp, err := currentLevelRuntime(w)
	if err != nil || runtimeComp == nil || runtimeComp.Level == nil {
		return err
	}

	for layerIndex := range runtimeComp.Level.Layers {
		layerName := runtimeLayerName(runtimeComp.Level, layerIndex)
		if layerName == "" {
			continue
		}

		active, ok := stateMap.States[levelLayerStateKey(runtimeComp.Name, layerName)]
		if !ok {
			continue
		}

		if err := SetLevelLayerActive(w, layerName, active); err != nil {
			return err
		}
	}

	return nil
}

func SetLevelLayerActive(world *ecs.World, layerName string, active bool) error {
	runtimeComp, err := currentLevelRuntime(world)
	if err != nil {
		return err
	}
	if runtimeComp.Level == nil {
		return fmt.Errorf("level runtime data not found")
	}

	layerIndex := findLevelLayerIndex(runtimeComp.Level, layerName)
	if layerIndex < 0 {
		return fmt.Errorf("level layer %q not found", layerName)
	}

	ensureLoadedLayerCapacity(runtimeComp, len(runtimeComp.Level.Layers))
	ensureLayerMeta(runtimeComp.Level, layerIndex)
	activeValue := active
	runtimeComp.Level.LayerMeta[layerIndex].Active = &activeValue

	if active && !runtimeComp.LoadedLayers[layerIndex] {
		tileSize := runtimeComp.TileSize
		if tileSize <= 0 {
			tileSize = 32
		}
		if err := levelentity.LoadLevelLayerToWorld(world, runtimeComp.Level, layerIndex, tileSize); err != nil {
			return err
		}
		runtimeComp.LoadedLayers[layerIndex] = true
	}

	setRuntimeLayerEntityState(world, layerIndex, active)
	if levelLayerHasPhysics(runtimeComp.Level, layerIndex) {
		tileSize := runtimeComp.TileSize
		if tileSize <= 0 {
			tileSize = 32
		}
		if err := levelentity.RebuildMergedLevelPhysics(world, runtimeComp.Level, tileSize); err != nil {
			return err
		}
	}
	if err := rebuildLevelGrid(world, runtimeComp); err != nil {
		return err
	}

	RecordLevelLayerState(world, layerName, active)
	return nil
}

func currentLevelRuntime(world *ecs.World) (*component.LevelRuntime, error) {
	if world == nil {
		return nil, fmt.Errorf("world is nil")
	}

	ent, ok := ecs.First(world, component.LevelRuntimeComponent.Kind())
	if !ok {
		return nil, fmt.Errorf("level runtime component not found")
	}

	runtimeComp, ok := ecs.Get(world, ent, component.LevelRuntimeComponent.Kind())
	if !ok || runtimeComp == nil {
		return nil, fmt.Errorf("level runtime component not found")
	}

	return runtimeComp, nil
}

func ensureLoadedLayerCapacity(runtimeComp *component.LevelRuntime, layerCount int) {
	if runtimeComp == nil || len(runtimeComp.LoadedLayers) >= layerCount {
		return
	}
	loadedLayers := make([]bool, layerCount)
	copy(loadedLayers, runtimeComp.LoadedLayers)
	runtimeComp.LoadedLayers = loadedLayers
}

func ensureLayerMeta(lvl *levels.Level, layerIndex int) {
	if lvl == nil || layerIndex < 0 {
		return
	}
	if len(lvl.LayerMeta) > layerIndex {
		return
	}
	meta := make([]levels.LayerMeta, layerIndex+1)
	copy(meta, lvl.LayerMeta)
	lvl.LayerMeta = meta
}

func setRuntimeLayerEntityState(world *ecs.World, layerIndex int, active bool) {
	disabled := !active
	ecs.ForEach(world, component.EntityLayerComponent.Kind(), func(e ecs.Entity, layer *component.EntityLayer) {
		if layer == nil || layer.Index != layerIndex {
			return
		}
		if sprite, ok := ecs.Get(world, e, component.SpriteComponent.Kind()); ok && sprite != nil {
			sprite.Disabled = disabled
		}
		if body, ok := ecs.Get(world, e, component.PhysicsBodyComponent.Kind()); ok && body != nil {
			body.Disabled = disabled
		}
		if hazard, ok := ecs.Get(world, e, component.HazardComponent.Kind()); ok && hazard != nil {
			hazard.Disabled = disabled
		}
		if circle, ok := ecs.Get(world, e, component.CircleRenderComponent.Kind()); ok && circle != nil {
			circle.Disabled = disabled
		}
		if input, ok := ecs.Get(world, e, component.InputComponent.Kind()); ok && input != nil {
			input.Disabled = disabled
		}
	})
	if batchEntity, ok := ecs.First(world, component.LevelGridComponent.Kind()); ok {
		if batchState, ok := ecs.Get(world, batchEntity, component.StaticTileBatchStateComponent.Kind()); ok && batchState != nil {
			batchState.Dirty = true
		}
	}
}

func rebuildLevelGrid(world *ecs.World, runtimeComp *component.LevelRuntime) error {
	if world == nil || runtimeComp == nil {
		return nil
	}

	gridEntity, ok := ecs.First(world, component.LevelGridComponent.Kind())
	if !ok {
		return nil
	}
	grid, ok := ecs.Get(world, gridEntity, component.LevelGridComponent.Kind())
	if !ok || grid == nil {
		return nil
	}

	tileSize := runtimeComp.TileSize
	if tileSize <= 0 {
		tileSize = grid.TileSize
	}
	if tileSize <= 0 {
		tileSize = 32
	}
	rebuilt := levelentity.BuildLevelGridData(runtimeComp.Level, tileSize)
	*grid = *rebuilt
	return nil
}

func findLevelLayerIndex(lvl *levels.Level, layerName string) int {
	if lvl == nil {
		return -1
	}
	needle := strings.TrimSpace(layerName)
	if needle == "" {
		return -1
	}
	for index := range lvl.LayerMeta {
		if strings.TrimSpace(lvl.LayerMeta[index].Name) == needle {
			return index
		}
	}
	return -1
}

func runtimeLayerName(lvl *levels.Level, layerIndex int) string {
	if lvl == nil || layerIndex < 0 || layerIndex >= len(lvl.LayerMeta) {
		return ""
	}
	return strings.TrimSpace(lvl.LayerMeta[layerIndex].Name)
}

func levelLayerHasPhysics(lvl *levels.Level, layerIndex int) bool {
	if lvl == nil || layerIndex < 0 || layerIndex >= len(lvl.LayerMeta) {
		return false
	}
	return lvl.LayerMeta[layerIndex].Physics
}
