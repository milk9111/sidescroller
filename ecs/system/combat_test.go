package system

import (
	"testing"

	"github.com/jakecoffman/cp"
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

func TestCombatThenKnockbackAppliesImpulseSameTick(t *testing.T) {
	w := ecs.NewWorld()
	scheduler := ecs.NewScheduler(NewCombatSystem(), NewDamageKnockbackSystem())

	attacker := ecs.CreateEntity(w)
	if err := ecs.Add(w, attacker, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add attacker player tag: %v", err)
	}
	if err := ecs.Add(w, attacker, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add attacker transform: %v", err)
	}
	if err := ecs.Add(w, attacker, component.AnimationComponent.Kind(), &component.Animation{Current: "attack", Frame: 5}); err != nil {
		t.Fatalf("add attacker animation: %v", err)
	}
	if err := ecs.Add(w, attacker, component.HitboxComponent.Kind(), &[]component.Hitbox{{Width: 70, Height: 24, OffsetX: 45, OffsetY: 32, Damage: 1, Anim: "attack", Frames: []int{5}}}); err != nil {
		t.Fatalf("add attacker hitbox: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.AITagComponent.Kind(), &component.AITag{}); err != nil {
		t.Fatalf("add target ai tag: %v", err)
	}
	if err := ecs.Add(w, target, component.KnockbackableComponent.Kind(), &component.Knockbackable{}); err != nil {
		t.Fatalf("add target knockbackable: %v", err)
	}
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 60, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}
	if err := ecs.Add(w, target, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 32, Height: 40, OffsetX: 31, OffsetY: 35}}); err != nil {
		t.Fatalf("add target hurtbox: %v", err)
	}
	if err := ecs.Add(w, target, component.HealthComponent.Kind(), &component.Health{Initial: 2, Current: 2}); err != nil {
		t.Fatalf("add target health: %v", err)
	}
	body := cp.NewBody(1, cp.MomentForBox(1, 32, 40))
	body.SetPosition(cp.Vector{X: 91, Y: 35})
	physicsBody := &component.PhysicsBody{Body: body, Width: 32, Height: 40, OffsetX: 31, OffsetY: 35, Mass: 1}
	if err := ecs.Add(w, target, component.PhysicsBodyComponent.Kind(), physicsBody); err != nil {
		t.Fatalf("add target physics body: %v", err)
	}

	scheduler.Update(w)

	if physicsBody.Body.Velocity().X <= 0 {
		t.Fatalf("expected same-tick knockback impulse to push target horizontally, got velocity %+v", physicsBody.Body.Velocity())
	}
	if _, ok := ecs.Get(w, target, component.DamageKnockbackRequestComponent.Kind()); ok {
		t.Fatal("expected knockback request to be consumed in the same scheduler tick")
	}
	if health, ok := ecs.Get(w, target, component.HealthComponent.Kind()); !ok || health == nil || health.Current != 1 {
		t.Fatalf("expected combat damage to apply before knockback, got health %+v", health)
	}
}

func TestCombatMarksPlayerHitsOnAIAsStrongKnockback(t *testing.T) {
	w := ecs.NewWorld()

	attacker := ecs.CreateEntity(w)
	if err := ecs.Add(w, attacker, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add attacker player tag: %v", err)
	}
	if err := ecs.Add(w, attacker, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add attacker transform: %v", err)
	}
	if err := ecs.Add(w, attacker, component.AnimationComponent.Kind(), &component.Animation{Current: "attack", Frame: 5}); err != nil {
		t.Fatalf("add attacker animation: %v", err)
	}
	if err := ecs.Add(w, attacker, component.HitboxComponent.Kind(), &[]component.Hitbox{{Width: 70, Height: 24, OffsetX: 45, OffsetY: 32, Damage: 1, Anim: "attack", Frames: []int{5}}}); err != nil {
		t.Fatalf("add attacker hitbox: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.AITagComponent.Kind(), &component.AITag{}); err != nil {
		t.Fatalf("add target ai tag: %v", err)
	}
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 60, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}
	if err := ecs.Add(w, target, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 32, Height: 40, OffsetX: 31, OffsetY: 35}}); err != nil {
		t.Fatalf("add target hurtbox: %v", err)
	}
	if err := ecs.Add(w, target, component.HealthComponent.Kind(), &component.Health{Initial: 2, Current: 2}); err != nil {
		t.Fatalf("add target health: %v", err)
	}

	NewCombatSystem().Update(w)

	req, ok := ecs.Get(w, target, component.DamageKnockbackRequestComponent.Kind())
	if !ok || req == nil {
		t.Fatal("expected combat to enqueue knockback for AI target")
	}
	if !req.Strong {
		t.Fatal("expected player hit on AI target to use strong knockback")
	}
}
