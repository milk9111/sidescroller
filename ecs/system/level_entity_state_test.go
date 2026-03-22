package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func addTestLevelRuntime(t *testing.T, w *ecs.World, name string) {
	t.Helper()

	ent := ecs.CreateEntity(w)
	if err := ecs.Add(w, ent, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{Name: name}); err != nil {
		t.Fatalf("add level runtime: %v", err)
	}
}

func addTestPlayerStateMap(t *testing.T, w *ecs.World) (*component.LevelEntityStateMap, ecs.Entity) {
	t.Helper()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 32, Height: 32}); err != nil {
		t.Fatalf("add player body: %v", err)
	}

	stateMap := &component.LevelEntityStateMap{States: map[string]component.PersistedLevelEntityState{}}
	if err := ecs.Add(w, player, component.LevelEntityStateMapComponent.Kind(), stateMap); err != nil {
		t.Fatalf("add level entity state map: %v", err)
	}

	return stateMap, player
}

func TestCombatRecordsDefeatedEnemyInPlayerStateMap(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)

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

	enemy := ecs.CreateEntity(w)
	if err := ecs.Add(w, enemy, component.AITagComponent.Kind(), &component.AITag{}); err != nil {
		t.Fatalf("add enemy ai tag: %v", err)
	}
	if err := ecs.Add(w, enemy, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "enemy_1"}); err != nil {
		t.Fatalf("add enemy game id: %v", err)
	}
	if err := ecs.Add(w, enemy, component.TransformComponent.Kind(), &component.Transform{X: 60, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add enemy transform: %v", err)
	}
	if err := ecs.Add(w, enemy, component.HurtboxComponent.Kind(), &[]component.Hurtbox{{Width: 32, Height: 40, OffsetX: 31, OffsetY: 35}}); err != nil {
		t.Fatalf("add enemy hurtbox: %v", err)
	}
	if err := ecs.Add(w, enemy, component.HealthComponent.Kind(), &component.Health{Initial: 1, Current: 1}); err != nil {
		t.Fatalf("add enemy health: %v", err)
	}

	NewCombatSystem().Update(w)

	if got := stateMap.States[levelEntityStateKey("disposal_1.json", "enemy_1")]; got != component.PersistedLevelEntityStateDefeated {
		t.Fatalf("expected enemy defeat state to be recorded, got %q", got)
	}
	if health, ok := ecs.Get(w, enemy, component.HealthComponent.Kind()); !ok || health == nil || health.Current != 0 {
		t.Fatalf("expected enemy health to reach zero, got %+v", health)
	}
}

func TestPickupCollectRecordsCollectedPickupInPlayerStateMap(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)

	pickup := ecs.CreateEntity(w)
	if err := ecs.Add(w, pickup, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "pickup_1"}); err != nil {
		t.Fatalf("add pickup game id: %v", err)
	}
	if err := ecs.Add(w, pickup, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add pickup transform: %v", err)
	}
	if err := ecs.Add(w, pickup, component.PickupComponent.Kind(), &component.Pickup{CollisionWidth: 24, CollisionHeight: 24}); err != nil {
		t.Fatalf("add pickup component: %v", err)
	}

	NewPickupCollectSystem().Update(w)

	if got := stateMap.States[levelEntityStateKey("disposal_1.json", "pickup_1")]; got != component.PersistedLevelEntityStateCollected {
		t.Fatalf("expected pickup collected state to be recorded, got %q", got)
	}
	if _, ok := ecs.Get(w, pickup, component.PickupComponent.Kind()); ok {
		t.Fatal("expected pickup behavior to be removed after collection")
	}
	if ttl, ok := ecs.Get(w, pickup, component.TTLComponent.Kind()); !ok || ttl == nil || ttl.Frames != 2 {
		t.Fatalf("expected pickup cleanup ttl to be scheduled, got %+v", ttl)
	}
}

func TestTriggerSystemRecordsUsedTriggerInPlayerStateMap(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)

	triggerEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, triggerEntity, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "trigger_1"}); err != nil {
		t.Fatalf("add trigger game id: %v", err)
	}
	if err := ecs.Add(w, triggerEntity, component.TransformComponent.Kind(), &component.Transform{}); err != nil {
		t.Fatalf("add trigger transform: %v", err)
	}
	if err := ecs.Add(w, triggerEntity, component.TriggerComponent.Kind(), &component.Trigger{Name: "once", Bounds: component.AABB{W: 32, H: 32}}); err != nil {
		t.Fatalf("add trigger component: %v", err)
	}

	NewTriggerSystem().Update(w)

	if got := stateMap.States[levelEntityStateKey("disposal_1.json", "trigger_1")]; got != component.PersistedLevelEntityStateUsed {
		t.Fatalf("expected trigger used state to be recorded, got %q", got)
	}
	trigger, ok := ecs.Get(w, triggerEntity, component.TriggerComponent.Kind())
	if !ok || trigger == nil || !trigger.Disabled {
		t.Fatalf("expected trigger to disable after use, got %+v", trigger)
	}
}

func TestApplyPersistedLevelEntityStatesRemovesDefeatedEnemiesAndCollectedPickupsAndDisablesUsedTriggers(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)
	stateMap.States[levelEntityStateKey("disposal_1.json", "enemy_1")] = component.PersistedLevelEntityStateDefeated
	stateMap.States[levelEntityStateKey("disposal_1.json", "pickup_1")] = component.PersistedLevelEntityStateCollected
	stateMap.States[levelEntityStateKey("disposal_1.json", "trigger_1")] = component.PersistedLevelEntityStateUsed

	enemy := ecs.CreateEntity(w)
	if err := ecs.Add(w, enemy, component.AITagComponent.Kind(), &component.AITag{}); err != nil {
		t.Fatalf("add enemy ai tag: %v", err)
	}
	if err := ecs.Add(w, enemy, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "enemy_1"}); err != nil {
		t.Fatalf("add enemy game id: %v", err)
	}

	pickup := ecs.CreateEntity(w)
	if err := ecs.Add(w, pickup, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "pickup_1"}); err != nil {
		t.Fatalf("add pickup game id: %v", err)
	}
	if err := ecs.Add(w, pickup, component.PickupComponent.Kind(), &component.Pickup{}); err != nil {
		t.Fatalf("add pickup component: %v", err)
	}

	triggerEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, triggerEntity, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "trigger_1"}); err != nil {
		t.Fatalf("add trigger game id: %v", err)
	}
	if err := ecs.Add(w, triggerEntity, component.TriggerComponent.Kind(), &component.Trigger{}); err != nil {
		t.Fatalf("add trigger component: %v", err)
	}

	applyPersistedLevelEntityStates(w)

	if ecs.IsAlive(w, enemy) {
		t.Fatal("expected defeated enemy to be removed on load")
	}
	if ecs.IsAlive(w, pickup) {
		t.Fatal("expected collected pickup to be removed on load")
	}
	trigger, ok := ecs.Get(w, triggerEntity, component.TriggerComponent.Kind())
	if !ok || trigger == nil || !trigger.Disabled {
		t.Fatalf("expected used trigger to load disabled, got %+v", trigger)
	}
}
