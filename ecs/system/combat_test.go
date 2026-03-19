package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestCombatDoesNotBlockHitWhenHurtboxStartsBeforeStaticBody(t *testing.T) {
	w := ecs.NewWorld()
	attacker := ecs.CreateEntity(w)
	if err := ecs.Add(w, attacker, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add attacker transform: %v", err)
	}
	if err := ecs.Add(w, attacker, component.AnimationComponent.Kind(), &component.Animation{}); err != nil {
		t.Fatalf("add attacker animation: %v", err)
	}
	if err := ecs.Add(w, attacker, component.HitboxComponent.Kind(), &[]component.Hitbox{{Width: 90, Height: 12, OffsetX: 45, Damage: 1}}); err != nil {
		t.Fatalf("add attacker hitbox: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 55, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}
	if err := ecs.Add(w, target, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 40, Height: 12, OffsetX: 0}}); err != nil {
		t.Fatalf("add target hurtbox: %v", err)
	}
	if err := ecs.Add(w, target, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Static: true, Width: 10, Height: 12, OffsetX: 0}); err != nil {
		t.Fatalf("add target body: %v", err)
	}
	if err := ecs.Add(w, target, component.HealthComponent.Kind(), &component.Health{Initial: 1, Current: 1}); err != nil {
		t.Fatalf("add target health: %v", err)
	}

	NewCombatSystem().Update(w)

	health, ok := ecs.Get(w, target, component.HealthComponent.Kind())
	if !ok || health == nil {
		t.Fatal("expected target health component")
	}
	if health.Current != 0 {
		t.Fatalf("expected front-extended hurtbox hit to land before static body, got health %d", health.Current)
	}
	if !hasHitTarget(w, attacker, target) {
		t.Fatal("expected attacker hitbox to record the target hit")
	}
}

func TestCombatBlocksHitWhenStaticBodyStartsBeforeHurtbox(t *testing.T) {
	w := ecs.NewWorld()
	attacker := ecs.CreateEntity(w)
	if err := ecs.Add(w, attacker, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add attacker transform: %v", err)
	}
	if err := ecs.Add(w, attacker, component.AnimationComponent.Kind(), &component.Animation{}); err != nil {
		t.Fatalf("add attacker animation: %v", err)
	}
	if err := ecs.Add(w, attacker, component.HitboxComponent.Kind(), &[]component.Hitbox{{Width: 90, Height: 12, OffsetX: 45, Damage: 1}}); err != nil {
		t.Fatalf("add attacker hitbox: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 55, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}
	if err := ecs.Add(w, target, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 20, Height: 12, OffsetX: 10}}); err != nil {
		t.Fatalf("add target hurtbox: %v", err)
	}
	if err := ecs.Add(w, target, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Static: true, Width: 20, Height: 12, OffsetX: -10}); err != nil {
		t.Fatalf("add target body: %v", err)
	}
	if err := ecs.Add(w, target, component.HealthComponent.Kind(), &component.Health{Initial: 1, Current: 1}); err != nil {
		t.Fatalf("add target health: %v", err)
	}

	NewCombatSystem().Update(w)

	health, ok := ecs.Get(w, target, component.HealthComponent.Kind())
	if !ok || health == nil {
		t.Fatal("expected target health component")
	}
	if health.Current != 1 {
		t.Fatalf("expected static body in front of hurtbox to block the hit, got health %d", health.Current)
	}
	if hasHitTarget(w, attacker, target) {
		t.Fatal("expected blocked hit to not record the target")
	}
}

func hasHitTarget(w *ecs.World, attacker, target ecs.Entity) bool {
	hitboxes, ok := ecs.Get(w, attacker, component.HitboxComponent.Kind())
	if !ok || hitboxes == nil || len(*hitboxes) == 0 {
		return false
	}
	return (*hitboxes)[0].HitTargets[uint64(target)]
}
