package system

import (
	"math"
	"testing"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestDamageKnockbackUsesAwayFromSourceForStrongPlayerHitsOnAI(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: -16, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.AITagComponent.Kind(), &component.AITag{}); err != nil {
		t.Fatalf("add target ai tag: %v", err)
	}
	if err := ecs.Add(w, target, component.KnockbackableComponent.Kind(), &component.Knockbackable{}); err != nil {
		t.Fatalf("add target knockbackable: %v", err)
	}
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}

	body := cp.NewBody(1, cp.MomentForBox(1, 32, 32))
	body.SetPosition(cp.Vector{X: 0, Y: 0})
	body.SetVelocityVector(cp.Vector{X: 0, Y: 4})
	physicsBody := &component.PhysicsBody{Body: body, Width: 32, Height: 32, Mass: 1}
	if err := ecs.Add(w, target, component.PhysicsBodyComponent.Kind(), physicsBody); err != nil {
		t.Fatalf("add target physics body: %v", err)
	}

	if err := ecs.Add(w, target, component.DamageKnockbackRequestComponent.Kind(), &component.DamageKnockback{
		SourceX:      0,
		SourceY:      -16,
		Strong:       true,
		SourceEntity: uint64(player),
	}); err != nil {
		t.Fatalf("add knockback request: %v", err)
	}

	NewDamageKnockbackSystem().Update(w)

	velocity := physicsBody.Body.Velocity()
	if math.Abs(velocity.X) > 1e-6 {
		t.Fatalf("expected vertical knockback away from the player, got velocity %+v", velocity)
	}
	if velocity.Y <= 4 {
		t.Fatalf("expected AI to be knocked away from the player instead of opposite its velocity, got velocity %+v", velocity)
	}
	if _, ok := ecs.Get(w, target, component.DamageKnockbackRequestComponent.Kind()); ok {
		t.Fatal("expected knockback request to be consumed")
	}
}
