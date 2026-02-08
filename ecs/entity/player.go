package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewPlayer(w *ecs.World) (ecs.Entity, error) {
	playerSpec, err := prefabs.LoadPlayerSpec()
	if err != nil {
		return 0, fmt.Errorf("player: load spec: %w", err)
	}

	entity := w.CreateEntity()

	if err := ecs.Add(w, entity, component.PlayerTagComponent, component.PlayerTag{}); err != nil {
		return 0, fmt.Errorf("player: add player tag: %w", err)
	}

	if err := ecs.Add(w, entity, component.PlayerComponent, component.Player{
		MoveSpeed: playerSpec.MoveSpeed,
		JumpSpeed: playerSpec.JumpSpeed,
	}); err != nil {
		return 0, fmt.Errorf("player: add player component: %w", err)
	}

	if err := ecs.Add(w, entity, component.InputComponent, component.Input{}); err != nil {
		return 0, fmt.Errorf("player: add input: %w", err)
	}

	if err := ecs.Add(w, entity, component.PlayerStateMachineComponent, component.PlayerStateMachine{}); err != nil {
		return 0, fmt.Errorf("player: add state machine: %w", err)
	}

	playerTransform := component.Transform{
		X:        playerSpec.Transform.X,
		Y:        playerSpec.Transform.Y,
		ScaleX:   playerSpec.Transform.ScaleX,
		ScaleY:   playerSpec.Transform.ScaleY,
		Rotation: playerSpec.Transform.Rotation,
	}
	if err := ecs.Add(
		w,
		entity,
		component.TransformComponent,
		playerTransform,
	); err != nil {
		return 0, fmt.Errorf("player: add transform: %w", err)
	}

	if err := ecs.Add(
		w,
		entity,
		component.SpriteComponent,
		component.Sprite{
			UseSource: playerSpec.Sprite.UseSource,
			OriginX:   playerSpec.Sprite.OriginX,
			OriginY:   playerSpec.Sprite.OriginY,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add sprite: %w", err)
	}

	spriteSheet, err := assets.LoadImage(playerSpec.Animation.Sheet)
	if err != nil {
		return 0, fmt.Errorf("player: load sprite sheet: %w", err)
	}

	defs := make(map[string]component.AnimationDef, len(playerSpec.Animation.Defs))
	for name, defSpec := range playerSpec.Animation.Defs {
		defs[name] = component.AnimationDef{
			Row:        defSpec.Row,
			ColStart:   defSpec.ColStart,
			FrameCount: defSpec.FrameCount,
			FrameW:     defSpec.FrameW,
			FrameH:     defSpec.FrameH,
			FPS:        defSpec.FPS,
			Loop:       defSpec.Loop,
		}
	}

	if err := ecs.Add(
		w,
		entity,
		component.AnimationComponent,
		component.Animation{
			Sheet:      spriteSheet,
			Defs:       defs,
			Current:    playerSpec.Animation.Current,
			Frame:      0,
			FrameTimer: 0,
			Playing:    true,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add animation: %w", err)
	}

	width := 0.0
	height := 0.0
	if def, ok := defs[playerSpec.Animation.Current]; ok {
		width = float64(def.FrameW)
		height = float64(def.FrameH)
	} else {
		for _, def := range defs {
			width = float64(def.FrameW)
			height = float64(def.FrameH)
			break
		}
	}
	if playerTransform.ScaleX == 0 {
		playerTransform.ScaleX = 1
	}
	if playerTransform.ScaleY == 0 {
		playerTransform.ScaleY = 1
	}
	width *= playerTransform.ScaleX
	height *= playerTransform.ScaleY
	if width == 0 {
		width = 32
	}
	if height == 0 {
		height = 32
	}

	if err := ecs.Add(
		w,
		entity,
		component.PhysicsBodyComponent,
		component.PhysicsBody{
			Width:        width,
			Height:       height,
			Mass:         1,
			Friction:     0.9,
			Elasticity:   0,
			AlignTopLeft: true,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add physics body: %w", err)
	}

	return entity, nil
}

func NewPlayerAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	entity, err := NewPlayer(w)
	if err != nil {
		return 0, err
	}
	transform, ok := ecs.Get(w, entity, component.TransformComponent)
	if !ok {
		transform = component.Transform{ScaleX: 1, ScaleY: 1}
	}
	transform.X = x
	transform.Y = y
	if err := ecs.Add(w, entity, component.TransformComponent, transform); err != nil {
		return 0, fmt.Errorf("player: override transform: %w", err)
	}
	return entity, nil
}
