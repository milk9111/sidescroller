package savegame

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestCaptureWorldAndApplyWorldRoundTrip(t *testing.T) {
	source := ecs.NewWorld()
	level := ecs.CreateEntity(source)
	_ = ecs.Add(source, level, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{Name: "disposal_1.json"})

	player := ecs.CreateEntity(source)
	_ = ecs.Add(source, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(source, player, component.HealthComponent.Kind(), &component.Health{Initial: 4, Current: 3})
	_ = ecs.Add(source, player, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{HealUses: 1})
	_ = ecs.Add(source, player, component.TransformComponent.Kind(), &component.Transform{X: 12, Y: 34, ScaleX: 1, ScaleY: 1, Rotation: 0.5})
	_ = ecs.Add(source, player, component.SpriteComponent.Kind(), &component.Sprite{FacingLeft: true})
	_ = ecs.Add(source, player, component.SafeRespawnComponent.Kind(), &component.SafeRespawn{X: 88, Y: 99, Initialized: true})
	_ = ecs.Add(source, player, component.PlayerCheckpointComponent.Kind(), &component.PlayerCheckpoint{Level: "disposal_1.json", X: 80, Y: 96, FacingLeft: true, Health: 4, HealUses: 0, Initialized: true})
	_ = ecs.Add(source, player, component.TransitionCooldownComponent.Kind(), &component.TransitionCooldown{Active: true, TransitionID: "left", TransitionIDs: []string{"left", "top_left"}})
	_ = ecs.Add(source, player, component.TransitionPopComponent.Kind(), &component.TransitionPop{VX: 3.5, VY: -8, FacingLeft: true, WallJumpDur: 6, WallJumpX: -32})
	_ = ecs.Add(source, player, component.InventoryComponent.Kind(), &component.Inventory{Items: []component.InventoryItem{{Prefab: "item_wrench.yaml", Count: 1}, {Prefab: "item_gear.yaml", Count: 3}}})
	_ = ecs.Add(source, player, component.LevelLayerStateMapComponent.Kind(), &component.LevelLayerStateMap{States: map[string]bool{"disposal_1.json#hidden_spikes": false}})
	_ = ecs.Add(source, player, component.LevelEntityStateMapComponent.Kind(), &component.LevelEntityStateMap{States: map[string]component.PersistedLevelEntityState{
		"disposal_1.json#trigger_1": component.PersistedLevelEntityStateUsed,
		"disposal_1.json#enemy_1":   component.PersistedLevelEntityStateDefeated,
	}})

	abilities := ecs.CreateEntity(source)
	_ = ecs.Add(source, abilities, component.AbilitiesComponent.Kind(), &component.Abilities{DoubleJump: true, Anchor: true})
	gears := ecs.CreateEntity(source)
	_ = ecs.Add(source, gears, component.PlayerGearCountComponent.Kind(), &component.PlayerGearCount{Count: 7})

	snapshot, err := CaptureWorld(source)
	if err != nil {
		t.Fatalf("capture world: %v", err)
	}

	target := ecs.NewWorld()
	level2 := ecs.CreateEntity(target)
	_ = ecs.Add(target, level2, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{Name: "disposal_1.json"})
	player2 := ecs.CreateEntity(target)
	_ = ecs.Add(target, player2, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(target, player2, component.HealthComponent.Kind(), &component.Health{Initial: 1, Current: 1})
	_ = ecs.Add(target, player2, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{HealUses: 2})
	_ = ecs.Add(target, player2, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(target, player2, component.SpriteComponent.Kind(), &component.Sprite{})

	if err := ApplyWorld(target, snapshot); err != nil {
		t.Fatalf("apply world: %v", err)
	}

	health, _ := ecs.Get(target, player2, component.HealthComponent.Kind())
	if health == nil || health.Initial != 4 || health.Current != 3 {
		t.Fatalf("unexpected health %+v", health)
	}
	tf, _ := ecs.Get(target, player2, component.TransformComponent.Kind())
	if tf == nil || tf.X != 12 || tf.Y != 34 || tf.Rotation != 0.5 {
		t.Fatalf("unexpected transform %+v", tf)
	}
	sprite, _ := ecs.Get(target, player2, component.SpriteComponent.Kind())
	if sprite == nil || !sprite.FacingLeft {
		t.Fatalf("unexpected sprite %+v", sprite)
	}
	safe, _ := ecs.Get(target, player2, component.SafeRespawnComponent.Kind())
	if safe == nil || !safe.Initialized || safe.X != 88 || safe.Y != 99 {
		t.Fatalf("unexpected safe respawn %+v", safe)
	}
	checkpoint, _ := ecs.Get(target, player2, component.PlayerCheckpointComponent.Kind())
	if checkpoint == nil || !checkpoint.Initialized || checkpoint.Level != "disposal_1.json" || checkpoint.X != 80 || checkpoint.Y != 96 || !checkpoint.FacingLeft || checkpoint.Health != 4 || checkpoint.HealUses != 0 {
		t.Fatalf("unexpected checkpoint %+v", checkpoint)
	}
	stateMachine, _ := ecs.Get(target, player2, component.PlayerStateMachineComponent.Kind())
	if stateMachine == nil || stateMachine.HealUses != 1 {
		t.Fatalf("unexpected player state machine %+v", stateMachine)
	}
	cooldown, _ := ecs.Get(target, player2, component.TransitionCooldownComponent.Kind())
	if cooldown == nil || !cooldown.Active || cooldown.TransitionID != "left" || len(cooldown.TransitionIDs) != 2 {
		t.Fatalf("unexpected transition cooldown %+v", cooldown)
	}
	pop, _ := ecs.Get(target, player2, component.TransitionPopComponent.Kind())
	if pop == nil || pop.VX != 3.5 || pop.VY != -8 || !pop.FacingLeft || pop.WallJumpDur != 6 || pop.WallJumpX != -32 {
		t.Fatalf("unexpected transition pop %+v", pop)
	}
	inventory, _ := ecs.Get(target, player2, component.InventoryComponent.Kind())
	if inventory == nil || len(inventory.Items) != 2 || inventory.Items[1].Count != 3 {
		t.Fatalf("unexpected inventory %+v", inventory)
	}
	layerStateMap, _ := ecs.Get(target, player2, component.LevelLayerStateMapComponent.Kind())
	if layerStateMap == nil || layerStateMap.States["disposal_1.json#hidden_spikes"] {
		t.Fatalf("unexpected layer state map %+v", layerStateMap)
	}
	stateMap, _ := ecs.Get(target, player2, component.LevelEntityStateMapComponent.Kind())
	if stateMap == nil || stateMap.States["disposal_1.json#trigger_1"] != component.PersistedLevelEntityStateUsed {
		t.Fatalf("unexpected state map %+v", stateMap)
	}

	abilitiesEntity, ok := ecs.First(target, component.AbilitiesComponent.Kind())
	if !ok {
		t.Fatal("expected abilities entity")
	}
	abilitiesComp, _ := ecs.Get(target, abilitiesEntity, component.AbilitiesComponent.Kind())
	if abilitiesComp == nil || !abilitiesComp.DoubleJump || !abilitiesComp.Anchor || abilitiesComp.WallGrab {
		t.Fatalf("unexpected abilities %+v", abilitiesComp)
	}
	persistent, _ := ecs.Get(target, abilitiesEntity, component.PersistentComponent.Kind())
	if persistent == nil || !persistent.KeepOnLevelChange || !persistent.KeepOnReload {
		t.Fatalf("expected abilities entity to persist across level changes and reloads, got %+v", persistent)
	}

	gearEntity, ok := ecs.First(target, component.PlayerGearCountComponent.Kind())
	if !ok {
		t.Fatal("expected gear count entity")
	}
	gearCount, _ := ecs.Get(target, gearEntity, component.PlayerGearCountComponent.Kind())
	if gearCount == nil || gearCount.Count != 7 {
		t.Fatalf("unexpected gear count %+v", gearCount)
	}
}
