package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestPlayerModulePositionReturnsPlayerTransform(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 42, Y: 96}); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	mod := PlayerModule().Build(w, nil, player, player)
	result, err := mod["position"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("position returned error: %v", err)
	}

	coords, ok := result.(*tengo.Array)
	if !ok || len(coords.Value) != 2 {
		t.Fatalf("position returned %T, want [x, y] array", result)
	}

	x, ok := coords.Value[0].(*tengo.Float)
	if !ok || x.Value != 42 {
		t.Fatalf("expected x=42, got %#v", coords.Value[0])
	}

	y, ok := coords.Value[1].(*tengo.Float)
	if !ok || y.Value != 96 {
		t.Fatalf("expected y=96, got %#v", coords.Value[1])
	}
}

func TestPlayerModuleWorldPositionUsesWorldCoordinatesForParentedPlayer(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 10, Y: 20, Parent: 1, WorldX: 144, WorldY: 288}); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	mod := PlayerModule().Build(w, nil, player, player)
	result, err := mod["world_position"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("world_position returned error: %v", err)
	}

	coords, ok := result.(*tengo.Array)
	if !ok || len(coords.Value) != 2 {
		t.Fatalf("world_position returned %T, want [x, y] array", result)
	}

	x, ok := coords.Value[0].(*tengo.Float)
	if !ok || x.Value != 144 {
		t.Fatalf("expected x=144, got %#v", coords.Value[0])
	}

	y, ok := coords.Value[1].(*tengo.Float)
	if !ok || y.Value != 288 {
		t.Fatalf("expected y=288, got %#v", coords.Value[1])
	}
}
