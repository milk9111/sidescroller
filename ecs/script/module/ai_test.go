package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestAIModuleFacePlayerUsesPhysicsBodyPosition(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerComponent.Kind(), &component.Player{}); err != nil {
		t.Fatalf("add player component: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 500, Y: 0}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	playerBody := cp.NewBody(1, cp.MomentForBox(1, 20, 40))
	playerBody.SetPosition(cp.Vector{X: 120, Y: 0})
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Body: playerBody, Width: 20, Height: 40, Mass: 1}); err != nil {
		t.Fatalf("add player physics body: %v", err)
	}

	enemy := ecs.CreateEntity(w)
	if err := ecs.Add(w, enemy, component.SpriteComponent.Kind(), &component.Sprite{FacingLeft: false}); err != nil {
		t.Fatalf("add enemy sprite: %v", err)
	}
	if err := ecs.Add(w, enemy, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0}); err != nil {
		t.Fatalf("add enemy transform: %v", err)
	}
	enemyBody := cp.NewBody(1, cp.MomentForBox(1, 104, 176))
	enemyBody.SetPosition(cp.Vector{X: 160, Y: 0})
	if err := ecs.Add(w, enemy, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Body: enemyBody, Width: 104, Height: 176, Mass: 1}); err != nil {
		t.Fatalf("add enemy physics body: %v", err)
	}

	mod := AIModule().Build(w, nil, enemy, enemy)
	result, err := mod["face_player"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("face_player returned error: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("face_player returned %v, want true", result)
	}

	sprite, ok := ecs.Get(w, enemy, component.SpriteComponent.Kind())
	if !ok || sprite == nil {
		t.Fatal("expected enemy sprite component")
	}
	if !sprite.FacingLeft {
		t.Fatal("expected enemy to face left based on physics body position")
	}
}