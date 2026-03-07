package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestSpawnPlayerAtLinkedTransitionFallsBackToReverseLink(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true}); err != nil {
		t.Fatalf("add player body: %v", err)
	}

	transition := ecs.CreateEntity(w)
	if err := ecs.Add(w, transition, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 800, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transition transform: %v", err)
	}
	if err := ecs.Add(w, transition, component.TransitionComponent.Kind(), &component.Transition{
		ID:       "e1",
		LinkedID: "right",
		EnterDir: component.TransitionDirRight,
		Bounds: component.AABB{
			W: 32,
			H: 128,
		},
	}); err != nil {
		t.Fatalf("add transition: %v", err)
	}

	p := &PersistenceSystem{}
	p.spawnPlayerAtLinkedTransition(w, component.LevelChangeRequest{
		SpawnTransitionID: "left",
		FromTransitionID:  "right",
	})

	transform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || transform == nil {
		t.Fatal("expected player transform")
	}
	if transform.X != 6 || transform.Y != 888 {
		t.Fatalf("expected player to spawn at fallback transition, got (%v,%v)", transform.X, transform.Y)
	}

	cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	if !ok || cooldown == nil {
		t.Fatal("expected transition cooldown")
	}
	if !cooldown.Active || cooldown.TransitionID != "e1" {
		t.Fatalf("expected cooldown on resolved transition e1, got active=%v id=%q", cooldown.Active, cooldown.TransitionID)
	}
}

func TestSpawnPlayerAtLinkedTransitionPrefersExactID(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true})

	fallback := ecs.CreateEntity(w)
	_ = ecs.Add(w, fallback, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 800, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, fallback, component.TransitionComponent.Kind(), &component.Transition{ID: "e1", LinkedID: "right", EnterDir: component.TransitionDirRight, Bounds: component.AABB{W: 32, H: 128}})

	exact := ecs.CreateEntity(w)
	_ = ecs.Add(w, exact, component.TransformComponent.Kind(), &component.Transform{X: 1248, Y: 32, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, exact, component.TransitionComponent.Kind(), &component.Transition{ID: "left", LinkedID: "something_else", EnterDir: component.TransitionDirLeft, Bounds: component.AABB{W: 32, H: 128}})

	p := &PersistenceSystem{}
	p.spawnPlayerAtLinkedTransition(w, component.LevelChangeRequest{
		SpawnTransitionID: "left",
		FromTransitionID:  "right",
	})

	transform, _ := ecs.Get(w, player, component.TransformComponent.Kind())
	if transform.X != 1254 || transform.Y != 120 {
		t.Fatalf("expected exact transition spawn, got (%v,%v)", transform.X, transform.Y)
	}

	cooldown, _ := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	if cooldown == nil || cooldown.TransitionID != "left" {
		t.Fatalf("expected cooldown id left, got %+v", cooldown)
	}
}

func TestResolvePersistentSingletonsCopiesLevelScopedComponents(t *testing.T) {
	w := ecs.NewWorld()
	kept := ecs.CreateEntity(w)
	if err := ecs.Add(w, kept, component.PersistentComponent.Kind(), &component.Persistent{ID: "player", KeepOnLevelChange: true}); err != nil {
		t.Fatalf("add kept persistent: %v", err)
	}
	if err := ecs.Add(w, kept, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 1}); err != nil {
		t.Fatalf("add kept layer: %v", err)
	}

	loaded := ecs.CreateEntity(w)
	if err := ecs.Add(w, loaded, component.PersistentComponent.Kind(), &component.Persistent{ID: "player", KeepOnLevelChange: true}); err != nil {
		t.Fatalf("add loaded persistent: %v", err)
	}
	if err := ecs.Add(w, loaded, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 4}); err != nil {
		t.Fatalf("add loaded layer: %v", err)
	}
	if err := ecs.Add(w, loaded, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "e3"}); err != nil {
		t.Fatalf("add loaded game entity id: %v", err)
	}

	p := &PersistenceSystem{}
	p.resolvePersistentSingletons(w, map[string]ecs.Entity{"player": kept})

	if ecs.IsAlive(w, loaded) {
		t.Fatal("expected duplicate loaded entity to be destroyed")
	}

	layer, ok := ecs.Get(w, kept, component.EntityLayerComponent.Kind())
	if !ok || layer == nil || layer.Index != 4 {
		t.Fatalf("expected kept entity layer to update to 4, got %+v", layer)
	}

	id, ok := ecs.Get(w, kept, component.GameEntityIDComponent.Kind())
	if !ok || id == nil || id.Value != "e3" {
		t.Fatalf("expected kept entity game id e3, got %+v", id)
	}
}
