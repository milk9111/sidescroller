package system

import (
	"testing"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
)

func TestCombatQueuesGlobalHitSignalInsteadOfBroadcastingToWatchers(t *testing.T) {
	w := ecs.NewWorld()

	attacker := ecs.CreateEntity(w)
	if err := ecs.Add(w, attacker, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "player"}); err != nil {
		t.Fatalf("add attacker game entity id: %v", err)
	}
	if err := ecs.Add(w, attacker, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add attacker transform: %v", err)
	}
	if err := ecs.Add(w, attacker, component.AnimationComponent.Kind(), &component.Animation{}); err != nil {
		t.Fatalf("add attacker animation: %v", err)
	}
	if err := ecs.Add(w, attacker, component.HitboxComponent.Kind(), &[]component.Hitbox{{Width: 40, Height: 20, OffsetX: 20, Damage: 1}}); err != nil {
		t.Fatalf("add attacker hitbox: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.ScriptComponent.Kind(), &component.Script{Path: "dummy.tengo"}); err != nil {
		t.Fatalf("add target script component: %v", err)
	}
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 25, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}
	if err := ecs.Add(w, target, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 20, Height: 20}}); err != nil {
		t.Fatalf("add target hurtbox: %v", err)
	}
	if err := ecs.Add(w, target, component.HealthComponent.Kind(), &component.Health{Initial: 2, Current: 2}); err != nil {
		t.Fatalf("add target health: %v", err)
	}

	watcher := ecs.CreateEntity(w)
	if err := ecs.Add(w, watcher, component.ScriptComponent.Kind(), &component.Script{Path: "dummy.tengo"}); err != nil {
		t.Fatalf("add watcher script component: %v", err)
	}

	NewCombatSystem().Update(w)

	targetQueue, ok := ecs.Get(w, target, component.ScriptSignalQueueComponent.Kind())
	if !ok || targetQueue == nil {
		t.Fatal("expected target to receive a direct hit signal")
	}
	if len(targetQueue.Events) != 1 {
		t.Fatalf("expected target to receive one hit event, got %#v", targetQueue.Events)
	}
	if targetQueue.Events[0].Name != "on_hit" {
		t.Fatalf("expected target hit event name on_hit, got %#v", targetQueue.Events[0])
	}
	if !targetQueue.Events[0].HasPosition {
		t.Fatal("expected target hit event to include contact position")
	}

	watcherQueue, ok := ecs.Get(w, watcher, component.ScriptSignalQueueComponent.Kind())
	if ok && watcherQueue != nil && len(watcherQueue.Events) > 0 {
		t.Fatalf("expected watcher to not receive per-entity broadcast hit events, got %#v", watcherQueue.Events)
	}

	globalQueueEntity, ok := ecs.First(w, component.GlobalHitSignalQueueComponent.Kind())
	if !ok {
		t.Fatal("expected global hit signal queue to exist")
	}
	globalQueue, ok := ecs.Get(w, globalQueueEntity, component.GlobalHitSignalQueueComponent.Kind())
	if !ok || globalQueue == nil {
		t.Fatal("expected global hit signal queue component")
	}
	if len(globalQueue.Events) != 1 {
		t.Fatalf("expected one global hit event, got %#v", globalQueue.Events)
	}
	if globalQueue.Events[0].Name != "on_hit" {
		t.Fatalf("expected global hit event name on_hit, got %#v", globalQueue.Events[0])
	}
	if globalQueue.Events[0].SourceGameEntity != "player" {
		t.Fatalf("expected global hit event source player, got %#v", globalQueue.Events[0])
	}
	if globalQueue.Events[0].ExcludedEntity != uint64(target) {
		t.Fatalf("expected global hit event to exclude target %d, got %#v", target, globalQueue.Events[0])
	}
}

func TestCombatClearsGlobalHitSignalQueueAtStartOfUpdate(t *testing.T) {
	w := ecs.NewWorld()

	queueEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, queueEntity, component.GlobalHitSignalQueueComponent.Kind(), &component.GlobalHitSignalQueue{Events: []component.ScriptSignalEvent{{Name: "on_hit", SourceGameEntity: "stale"}}}); err != nil {
		t.Fatalf("add stale global hit queue: %v", err)
	}

	attacker := ecs.CreateEntity(w)
	if err := ecs.Add(w, attacker, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "player"}); err != nil {
		t.Fatalf("add attacker game entity id: %v", err)
	}
	if err := ecs.Add(w, attacker, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add attacker transform: %v", err)
	}
	if err := ecs.Add(w, attacker, component.AnimationComponent.Kind(), &component.Animation{}); err != nil {
		t.Fatalf("add attacker animation: %v", err)
	}
	if err := ecs.Add(w, attacker, component.HitboxComponent.Kind(), &[]component.Hitbox{{Width: 40, Height: 20, OffsetX: 20, Damage: 1}}); err != nil {
		t.Fatalf("add attacker hitbox: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 25, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}
	if err := ecs.Add(w, target, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 20, Height: 20}}); err != nil {
		t.Fatalf("add target hurtbox: %v", err)
	}
	if err := ecs.Add(w, target, component.HealthComponent.Kind(), &component.Health{Initial: 2, Current: 2}); err != nil {
		t.Fatalf("add target health: %v", err)
	}

	NewCombatSystem().Update(w)

	globalQueue, ok := ecs.Get(w, queueEntity, component.GlobalHitSignalQueueComponent.Kind())
	if !ok || globalQueue == nil {
		t.Fatal("expected global hit signal queue component")
	}
	if len(globalQueue.Events) != 1 {
		t.Fatalf("expected stale global hit events to be cleared before queueing new hit, got %#v", globalQueue.Events)
	}
	if globalQueue.Events[0].SourceGameEntity != "player" {
		t.Fatalf("expected fresh hit event source player, got %#v", globalQueue.Events[0])
	}
}

func TestCombatBroadcastEnablesPlayerAttackHitEmitterPrefab(t *testing.T) {
	w := ecs.NewWorld()
	scheduler := ecs.NewScheduler(NewCombatSystem(), NewScriptSystem())

	attacker := ecs.CreateEntity(w)
	if err := ecs.Add(w, attacker, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "player"}); err != nil {
		t.Fatalf("add attacker game entity id: %v", err)
	}
	if err := ecs.Add(w, attacker, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add attacker transform: %v", err)
	}
	if err := ecs.Add(w, attacker, component.AnimationComponent.Kind(), &component.Animation{}); err != nil {
		t.Fatalf("add attacker animation: %v", err)
	}
	if err := ecs.Add(w, attacker, component.HitboxComponent.Kind(), &[]component.Hitbox{{Width: 40, Height: 20, OffsetX: 20, Damage: 1}}); err != nil {
		t.Fatalf("add attacker hitbox: %v", err)
	}

	target := ecs.CreateEntity(w)
	if err := ecs.Add(w, target, component.TransformComponent.Kind(), &component.Transform{X: 25, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add target transform: %v", err)
	}
	if err := ecs.Add(w, target, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 20, Height: 20}}); err != nil {
		t.Fatalf("add target hurtbox: %v", err)
	}
	if err := ecs.Add(w, target, component.HealthComponent.Kind(), &component.Health{Initial: 2, Current: 2}); err != nil {
		t.Fatalf("add target health: %v", err)
	}

	emitterEnt, err := entity.BuildEntity(w, "emitter_player_attack_hit.yaml")
	if err != nil {
		t.Fatalf("build player attack hit emitter prefab: %v", err)
	}

	emitter, ok := ecs.Get(w, emitterEnt, component.ParticleEmitterComponent.Kind())
	if !ok || emitter == nil {
		t.Fatal("expected emitter particle component")
	}
	if !emitter.Disabled {
		t.Fatal("expected emitter to start disabled")
	}

	scheduler.Update(w)

	if emitter.Disabled {
		t.Fatal("expected emitter to be enabled after broadcast hit")
	}

	tf, ok := ecs.Get(w, emitterEnt, component.TransformComponent.Kind())
	if !ok || tf == nil {
		t.Fatal("expected emitter transform component")
	}
	if tf.X == 0 && tf.Y == 0 {
		t.Fatalf("expected emitter transform to move to hit position, got (%v,%v)", tf.X, tf.Y)
	}
}

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
