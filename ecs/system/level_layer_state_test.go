package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/levels"
)

func TestApplyPersistedLevelLayerStatesDisablesCurrentLevelLayer(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.LevelLayerStateMapComponent.Kind(), &component.LevelLayerStateMap{States: map[string]bool{"disposal_1.json#Secret": false}}); err != nil {
		t.Fatalf("add layer state map: %v", err)
	}

	active := true
	runtimeEnt := ecs.CreateEntity(w)
	if err := ecs.Add(w, runtimeEnt, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{
		Name: "disposal_1.json",
		Level: &levels.Level{
			Layers:    [][]int{{0}, {0}},
			LayerMeta: []levels.LayerMeta{{Name: "Base", Active: &active}, {Name: "Secret", Active: &active}},
		},
		TileSize:     32,
		LoadedLayers: []bool{true, true},
	}); err != nil {
		t.Fatalf("add level runtime: %v", err)
	}

	gridEnt := ecs.CreateEntity(w)
	if err := ecs.Add(w, gridEnt, component.LevelGridComponent.Kind(), &component.LevelGrid{TileSize: 32}); err != nil {
		t.Fatalf("add level grid: %v", err)
	}
	if err := ecs.Add(w, gridEnt, component.StaticTileBatchStateComponent.Kind(), &component.StaticTileBatchState{}); err != nil {
		t.Fatalf("add static tile batch state: %v", err)
	}

	layerEnt := ecs.CreateEntity(w)
	if err := ecs.Add(w, layerEnt, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 1}); err != nil {
		t.Fatalf("add entity layer: %v", err)
	}
	if err := ecs.Add(w, layerEnt, component.SpriteComponent.Kind(), &component.Sprite{}); err != nil {
		t.Fatalf("add sprite: %v", err)
	}
	if err := ecs.Add(w, layerEnt, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{}); err != nil {
		t.Fatalf("add physics body: %v", err)
	}

	if err := applyPersistedLevelLayerStates(w); err != nil {
		t.Fatalf("apply persisted layer states: %v", err)
	}

	runtimeComp, _ := ecs.Get(w, runtimeEnt, component.LevelRuntimeComponent.Kind())
	if runtimeComp == nil || runtimeComp.Level == nil || runtimeComp.Level.LayerMeta[1].Active == nil || *runtimeComp.Level.LayerMeta[1].Active {
		t.Fatalf("expected secret layer to be inactive, got %+v", runtimeComp)
	}
	sprite, _ := ecs.Get(w, layerEnt, component.SpriteComponent.Kind())
	if sprite == nil || !sprite.Disabled {
		t.Fatalf("expected layer sprite to be disabled, got %+v", sprite)
	}
	body, _ := ecs.Get(w, layerEnt, component.PhysicsBodyComponent.Kind())
	if body == nil || !body.Disabled {
		t.Fatalf("expected layer body to be disabled, got %+v", body)
	}
	batchState, _ := ecs.Get(w, gridEnt, component.StaticTileBatchStateComponent.Kind())
	if batchState == nil || !batchState.Dirty {
		t.Fatalf("expected static tile batch state to be dirtied, got %+v", batchState)
	}
}
