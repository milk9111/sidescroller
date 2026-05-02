package system

import (
	"strings"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type ShrineSystem struct{}

func NewShrineSystem() *ShrineSystem {
	return &ShrineSystem{}
}

func (s *ShrineSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	pressed, _ := dialogueInputPressed(w)
	if !pressed {
		return
	}

	if _, ok := ecs.First(w, component.CheckpointReloadRequestComponent.Kind()); ok {
		return
	}
	if _, ok := ecs.First(w, component.EnemyRespawnRequestComponent.Kind()); ok {
		return
	}

	popupEntity, ok := ecs.First(w, component.ShrinePopupComponent.Kind())
	if !ok {
		return
	}
	popup, ok := ecs.Get(w, popupEntity, component.ShrinePopupComponent.Kind())
	if !ok || popup == nil || popup.TargetShrineEntity == 0 {
		return
	}
	sprite, ok := ecs.Get(w, popupEntity, component.SpriteComponent.Kind())
	if !ok || sprite == nil || sprite.Disabled {
		return
	}

	shrineEntity := ecs.Entity(popup.TargetShrineEntity)
	if !shrineEntity.Valid() || !ecs.IsAlive(w, shrineEntity) || !ecs.Has(w, shrineEntity, component.ShrineComponent.Kind()) {
		return
	}

	activateShrine(w)
}

func activateShrine(w *ecs.World) {
	if w == nil {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	transform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return
	}

	facingLeft := false
	if sprite, ok := ecs.Get(w, player, component.SpriteComponent.Kind()); ok && sprite != nil {
		facingLeft = sprite.FacingLeft
	}

	healthValue := 0
	if health, ok := ecs.Get(w, player, component.HealthComponent.Kind()); ok && health != nil {
		health.Current = health.Initial
		healthValue = health.Current
		_ = ecs.Add(w, player, component.HealthComponent.Kind(), health)
	}

	healUses := 0
	if stateMachine, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind()); ok && stateMachine != nil {
		stateMachine.HealUses = 0
		healUses = stateMachine.HealUses
		_ = ecs.Add(w, player, component.PlayerStateMachineComponent.Kind(), stateMachine)
	}

	levelName := currentCheckpointLevelName(w)
	checkpoint := &component.PlayerCheckpoint{
		Level:       levelName,
		X:           transform.X,
		Y:           transform.Y,
		FacingLeft:  facingLeft,
		Health:      healthValue,
		HealUses:    healUses,
		Initialized: true,
	}
	_ = ecs.Add(w, player, component.PlayerCheckpointComponent.Kind(), checkpoint)
	_ = ecs.Add(w, player, component.SafeRespawnComponent.Kind(), &component.SafeRespawn{X: transform.X, Y: transform.Y, Initialized: true})

	req := ecs.CreateEntity(w)
	_ = ecs.Add(w, req, component.EnemyRespawnRequestComponent.Kind(), &component.EnemyRespawnRequest{})
}

func currentCheckpointLevelName(w *ecs.World) string {
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

	return strings.TrimSpace(runtimeComp.Name)
}