package savegame

import (
	"fmt"
	"strings"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	playerAbilitiesPersistentID = "player_abilities"
	playerGearPersistentID      = "player_gears"
)

func CaptureWorld(w *ecs.World) (*File, error) {
	if w == nil {
		return nil, fmt.Errorf("capture save: nil world")
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return nil, fmt.Errorf("capture save: player entity not found")
	}

	levelName := currentLevelName(w)
	if strings.TrimSpace(levelName) == "" {
		return nil, fmt.Errorf("capture save: current level not found")
	}

	snapshot := &File{
		Version: CurrentVersion,
		Level:   levelName,
		Player: PlayerState{
			Inventory: cloneInventoryItems(nil),
		},
		LevelEntityStates: map[string]string{},
	}

	if health, ok := ecs.Get(w, player, component.HealthComponent.Kind()); ok && health != nil {
		snapshot.Player.Health = HealthState{Initial: health.Initial, Current: health.Current}
	}
	if stateMachine, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind()); ok && stateMachine != nil {
		snapshot.Player.HealUses = stateMachine.HealUses
	}
	if tf, ok := ecs.Get(w, player, component.TransformComponent.Kind()); ok && tf != nil {
		snapshot.Player.Transform = TransformState{
			X:        tf.X,
			Y:        tf.Y,
			ScaleX:   tf.ScaleX,
			ScaleY:   tf.ScaleY,
			Rotation: tf.Rotation,
		}
	}
	if safe, ok := ecs.Get(w, player, component.SafeRespawnComponent.Kind()); ok && safe != nil {
		snapshot.Player.SafeRespawn = SafeRespawnState{X: safe.X, Y: safe.Y, Initialized: safe.Initialized}
	}
	if checkpoint, ok := ecs.Get(w, player, component.PlayerCheckpointComponent.Kind()); ok && checkpoint != nil {
		snapshot.Player.Checkpoint = CheckpointState{
			Level:       checkpoint.Level,
			X:           checkpoint.X,
			Y:           checkpoint.Y,
			FacingLeft:  checkpoint.FacingLeft,
			Health:      checkpoint.Health,
			HealUses:    checkpoint.HealUses,
			Initialized: checkpoint.Initialized,
		}
	}
	if sprite, ok := ecs.Get(w, player, component.SpriteComponent.Kind()); ok && sprite != nil {
		snapshot.Player.FacingLeft = sprite.FacingLeft
	}
	if cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind()); ok && cooldown != nil && cooldown.Active {
		snapshot.Player.TransitionCooldown = &TransitionCooldownState{
			Active:        cooldown.Active,
			TransitionID:  cooldown.TransitionID,
			TransitionIDs: append([]string(nil), cooldown.TransitionIDs...),
		}
	}
	if pop, ok := ecs.Get(w, player, component.TransitionPopComponent.Kind()); ok && pop != nil {
		snapshot.Player.TransitionPop = &TransitionPopState{
			VX:          pop.VX,
			VY:          pop.VY,
			FacingLeft:  pop.FacingLeft,
			WallJumpDur: pop.WallJumpDur,
			WallJumpX:   pop.WallJumpX,
			Applied:     pop.Applied,
			Airborne:    pop.Airborne,
		}
	}
	if inventory, ok := ecs.Get(w, player, component.InventoryComponent.Kind()); ok && inventory != nil {
		snapshot.Player.Inventory = captureInventory(inventory)
	}
	if layerStateMap, ok := ecs.Get(w, player, component.LevelLayerStateMapComponent.Kind()); ok && layerStateMap != nil {
		snapshot.LevelLayerStates = captureLevelLayerStates(layerStateMap)
	}
	if stateMap, ok := ecs.Get(w, player, component.LevelEntityStateMapComponent.Kind()); ok && stateMap != nil {
		snapshot.LevelEntityStates = captureLevelEntityStates(stateMap)
	}

	if abilitiesEntity, ok := ecs.First(w, component.AbilitiesComponent.Kind()); ok {
		if abilities, ok := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind()); ok && abilities != nil {
			snapshot.Player.Abilities = AbilitiesState{
				DoubleJump: abilities.DoubleJump,
				WallGrab:   abilities.WallGrab,
				Anchor:     abilities.Anchor,
				Heal:       abilities.Heal,
			}
		}
	}

	if gearEntity, ok := ecs.First(w, component.PlayerGearCountComponent.Kind()); ok {
		if gearCount, ok := ecs.Get(w, gearEntity, component.PlayerGearCountComponent.Kind()); ok && gearCount != nil {
			snapshot.Player.GearCount = gearCount.Count
		}
	}

	return snapshot, nil
}

func ApplyWorld(w *ecs.World, file *File) error {
	if w == nil {
		return fmt.Errorf("apply save: nil world")
	}
	if file == nil {
		return nil
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return fmt.Errorf("apply save: player entity not found")
	}

	if health, ok := ecs.Get(w, player, component.HealthComponent.Kind()); ok && health != nil {
		health.Initial = file.Player.Health.Initial
		health.Current = file.Player.Health.Current
		_ = ecs.Add(w, player, component.HealthComponent.Kind(), health)
	} else if file.Player.Health.Initial > 0 || file.Player.Health.Current > 0 {
		_ = ecs.Add(w, player, component.HealthComponent.Kind(), &component.Health{
			Initial: file.Player.Health.Initial,
			Current: file.Player.Health.Current,
		})
	}

	if tf, ok := ecs.Get(w, player, component.TransformComponent.Kind()); ok && tf != nil {
		tf.X = file.Player.Transform.X
		tf.Y = file.Player.Transform.Y
		tf.ScaleX = file.Player.Transform.ScaleX
		tf.ScaleY = file.Player.Transform.ScaleY
		tf.Rotation = file.Player.Transform.Rotation
		_ = ecs.Add(w, player, component.TransformComponent.Kind(), tf)
	}

	if sprite, ok := ecs.Get(w, player, component.SpriteComponent.Kind()); ok && sprite != nil {
		sprite.FacingLeft = file.Player.FacingLeft
		_ = ecs.Add(w, player, component.SpriteComponent.Kind(), sprite)
	}

	if stateMachine, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind()); ok && stateMachine != nil {
		stateMachine.HealUses = file.Player.HealUses
		_ = ecs.Add(w, player, component.PlayerStateMachineComponent.Kind(), stateMachine)
	}

	_ = ecs.Add(w, player, component.SafeRespawnComponent.Kind(), &component.SafeRespawn{
		X:           file.Player.SafeRespawn.X,
		Y:           file.Player.SafeRespawn.Y,
		Initialized: file.Player.SafeRespawn.Initialized,
	})
	_ = ecs.Add(w, player, component.PlayerCheckpointComponent.Kind(), &component.PlayerCheckpoint{
		Level:       file.Player.Checkpoint.Level,
		X:           file.Player.Checkpoint.X,
		Y:           file.Player.Checkpoint.Y,
		FacingLeft:  file.Player.Checkpoint.FacingLeft,
		Health:      file.Player.Checkpoint.Health,
		HealUses:    file.Player.Checkpoint.HealUses,
		Initialized: file.Player.Checkpoint.Initialized,
	})
	if file.Player.TransitionCooldown != nil && file.Player.TransitionCooldown.Active {
		_ = ecs.Add(w, player, component.TransitionCooldownComponent.Kind(), &component.TransitionCooldown{
			Active:        file.Player.TransitionCooldown.Active,
			TransitionID:  file.Player.TransitionCooldown.TransitionID,
			TransitionIDs: append([]string(nil), file.Player.TransitionCooldown.TransitionIDs...),
		})
	}
	if file.Player.TransitionPop != nil {
		_ = ecs.Add(w, player, component.TransitionPopComponent.Kind(), &component.TransitionPop{
			VX:          file.Player.TransitionPop.VX,
			VY:          file.Player.TransitionPop.VY,
			FacingLeft:  file.Player.TransitionPop.FacingLeft,
			WallJumpDur: file.Player.TransitionPop.WallJumpDur,
			WallJumpX:   file.Player.TransitionPop.WallJumpX,
			Applied:     file.Player.TransitionPop.Applied,
			Airborne:    file.Player.TransitionPop.Airborne,
		})
	}
	_ = ecs.Add(w, player, component.InventoryComponent.Kind(), &component.Inventory{Items: applyInventory(file.Player.Inventory)})
	_ = ecs.Add(w, player, component.LevelLayerStateMapComponent.Kind(), &component.LevelLayerStateMap{States: applyLevelLayerStates(file.LevelLayerStates)})
	_ = ecs.Add(w, player, component.LevelEntityStateMapComponent.Kind(), &component.LevelEntityStateMap{States: applyLevelEntityStates(file.LevelEntityStates)})

	abilitiesEntity := ensureAbilitiesEntity(w)
	_ = ecs.Add(w, abilitiesEntity, component.AbilitiesComponent.Kind(), &component.Abilities{
		DoubleJump: file.Player.Abilities.DoubleJump,
		WallGrab:   file.Player.Abilities.WallGrab,
		Anchor:     file.Player.Abilities.Anchor,
		Heal:       file.Player.Abilities.Heal,
	})

	gearEntity := ensureGearCountEntity(w)
	_ = ecs.Add(w, gearEntity, component.PlayerGearCountComponent.Kind(), &component.PlayerGearCount{Count: file.Player.GearCount})

	return nil
}

func currentLevelName(w *ecs.World) string {
	if w == nil {
		return ""
	}

	ent, ok := ecs.First(w, component.LevelRuntimeComponent.Kind())
	if !ok {
		return ""
	}

	runtimeComp, ok := ecs.Get(w, ent, component.LevelRuntimeComponent.Kind())
	if !ok || runtimeComp == nil {
		return ""
	}

	return runtimeComp.Name
}

func ensureAbilitiesEntity(w *ecs.World) ecs.Entity {
	if ent, ok := ecs.First(w, component.AbilitiesComponent.Kind()); ok {
		return ent
	}

	ent := ecs.CreateEntity(w)
	_ = ecs.Add(w, ent, component.PersistentComponent.Kind(), &component.Persistent{
		ID:                playerAbilitiesPersistentID,
		KeepOnLevelChange: true,
		KeepOnReload:      true,
	})
	return ent
}

func ensureGearCountEntity(w *ecs.World) ecs.Entity {
	if ent, ok := ecs.First(w, component.PlayerGearCountComponent.Kind()); ok {
		return ent
	}

	ent := ecs.CreateEntity(w)
	_ = ecs.Add(w, ent, component.PersistentComponent.Kind(), &component.Persistent{
		ID:                playerGearPersistentID,
		KeepOnLevelChange: true,
		KeepOnReload:      false,
	})
	return ent
}

func captureInventory(inventory *component.Inventory) []InventoryItem {
	if inventory == nil || len(inventory.Items) == 0 {
		return nil
	}

	items := make([]InventoryItem, 0, len(inventory.Items))
	for _, item := range inventory.Items {
		items = append(items, InventoryItem{Prefab: item.Prefab, Count: item.Count})
	}
	return items
}

func applyInventory(items []InventoryItem) []component.InventoryItem {
	if len(items) == 0 {
		return nil
	}

	copy := make([]component.InventoryItem, 0, len(items))
	for _, item := range items {
		copy = append(copy, component.InventoryItem{Prefab: item.Prefab, Count: item.Count})
	}
	return copy
}

func captureLevelLayerStates(stateMap *component.LevelLayerStateMap) map[string]bool {
	if stateMap == nil || len(stateMap.States) == 0 {
		return map[string]bool{}
	}

	copy := make(map[string]bool, len(stateMap.States))
	for key, value := range stateMap.States {
		copy[key] = value
	}
	return copy
}

func applyLevelLayerStates(states map[string]bool) map[string]bool {
	if len(states) == 0 {
		return map[string]bool{}
	}

	copy := make(map[string]bool, len(states))
	for key, value := range states {
		copy[key] = value
	}
	return copy
}

func captureLevelEntityStates(stateMap *component.LevelEntityStateMap) map[string]string {
	if stateMap == nil || len(stateMap.States) == 0 {
		return map[string]string{}
	}

	copy := make(map[string]string, len(stateMap.States))
	for key, value := range stateMap.States {
		copy[key] = string(value)
	}
	return copy
}

func applyLevelEntityStates(states map[string]string) map[string]component.PersistedLevelEntityState {
	if len(states) == 0 {
		return map[string]component.PersistedLevelEntityState{}
	}

	copy := make(map[string]component.PersistedLevelEntityState, len(states))
	for key, value := range states {
		copy[key] = component.PersistedLevelEntityState(value)
	}
	return copy
}
