package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestEntityDestroyRecordsLevelStateBeforeDestroying(t *testing.T) {
	w := ecs.NewWorld()
	runtimeEnt := ecs.CreateEntity(w)
	if err := ecs.Add(w, runtimeEnt, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{Name: "disposal_1.json"}); err != nil {
		t.Fatalf("add level runtime: %v", err)
	}

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.LevelEntityStateMapComponent.Kind(), &component.LevelEntityStateMap{States: map[string]component.PersistedLevelEntityState{}}); err != nil {
		t.Fatalf("add state map: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "solid_tile_platform_1"}); err != nil {
		t.Fatalf("add target game id: %v", err)
	}

	moduleValues := EntityModule().Build(w, nil, 0, target)
	destroyFn, ok := moduleValues["destroy"].(*tengo.UserFunction)
	if !ok || destroyFn == nil {
		t.Fatal("expected destroy function")
	}

	result, err := destroyFn.Value()
	if err != nil {
		t.Fatalf("destroy call: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("expected destroy to return true, got %#v", result)
	}
	if ecs.IsAlive(w, target) {
		t.Fatal("expected target to be destroyed")
	}

	stateMap, _ := ecs.Get(w, player, component.LevelEntityStateMapComponent.Kind())
	if stateMap == nil || stateMap.States["disposal_1.json#solid_tile_platform_1"] != component.PersistedLevelEntityStateDefeated {
		t.Fatalf("expected destroyed entity state to be persisted, got %+v", stateMap)
	}
}
