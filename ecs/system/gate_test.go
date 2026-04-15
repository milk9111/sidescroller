package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestGateSystemRecordsAndRemovesGateWhenSpriteDisablesAfterBeingEnabled(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)

	gate := ecs.CreateEntity(w)
	if err := ecs.Add(w, gate, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "gate_1"}); err != nil {
		t.Fatalf("add gate game id: %v", err)
	}
	if err := ecs.Add(w, gate, component.GateComponent.Kind(), &component.Gate{}); err != nil {
		t.Fatalf("add gate component: %v", err)
	}
	if err := ecs.Add(w, gate, component.ArenaNodeComponent.Kind(), &component.ArenaNode{Active: true}); err != nil {
		t.Fatalf("add arena node: %v", err)
	}
	if err := ecs.Add(w, gate, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false}); err != nil {
		t.Fatalf("add sprite: %v", err)
	}
	if err := ecs.Add(w, gate, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Disabled: false, Width: 32, Height: 32}); err != nil {
		t.Fatalf("add physics body: %v", err)
	}

	system := NewGateSystem()
	system.Update(w)

	if !ecs.IsAlive(w, gate) {
		t.Fatal("expected gate to remain after initial enabled update")
	}

	sprite, ok := ecs.Get(w, gate, component.SpriteComponent.Kind())
	if !ok || sprite == nil {
		t.Fatal("expected gate sprite to exist")
	}
	sprite.Disabled = true

	system.Update(w)

	if ecs.IsAlive(w, gate) {
		t.Fatal("expected gate to be removed after sprite disabled transition")
	}
	if got := stateMap.States[levelEntityStateKey("disposal_1.json", "gate_1")]; got != component.PersistedLevelEntityStateUsed {
		t.Fatalf("expected gate used state to be recorded, got %q", got)
	}
}

func TestGateSystemIgnoresInitiallyDisabledGate(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)

	gate := ecs.CreateEntity(w)
	if err := ecs.Add(w, gate, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "gate_1"}); err != nil {
		t.Fatalf("add gate game id: %v", err)
	}
	if err := ecs.Add(w, gate, component.GateComponent.Kind(), &component.Gate{}); err != nil {
		t.Fatalf("add gate component: %v", err)
	}
	if err := ecs.Add(w, gate, component.ArenaNodeComponent.Kind(), &component.ArenaNode{Active: true}); err != nil {
		t.Fatalf("add arena node: %v", err)
	}
	if err := ecs.Add(w, gate, component.SpriteComponent.Kind(), &component.Sprite{Disabled: true}); err != nil {
		t.Fatalf("add sprite: %v", err)
	}
	if err := ecs.Add(w, gate, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Disabled: true, Width: 32, Height: 32}); err != nil {
		t.Fatalf("add physics body: %v", err)
	}

	NewGateSystem().Update(w)

	if !ecs.IsAlive(w, gate) {
		t.Fatal("expected initially disabled gate to remain alive")
	}
	if _, ok := stateMap.States[levelEntityStateKey("disposal_1.json", "gate_1")]; ok {
		t.Fatal("expected initially disabled gate not to be persisted as used")
	}
}

func TestApplyPersistedLevelEntityStatesRemovesUsedGates(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)
	stateMap.States[levelEntityStateKey("disposal_1.json", "gate_1")] = component.PersistedLevelEntityStateUsed

	gate := ecs.CreateEntity(w)
	if err := ecs.Add(w, gate, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "gate_1"}); err != nil {
		t.Fatalf("add gate game id: %v", err)
	}
	if err := ecs.Add(w, gate, component.GateComponent.Kind(), &component.Gate{}); err != nil {
		t.Fatalf("add gate component: %v", err)
	}

	applyPersistedLevelEntityStates(w)

	if ecs.IsAlive(w, gate) {
		t.Fatal("expected used gate to be removed on load")
	}
}
