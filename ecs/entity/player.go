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
	if err := ecs.Add(
		w,
		entity,
		component.TransformComponent,
		component.Transform{
			X:        playerSpec.Transform.X,
			Y:        playerSpec.Transform.Y,
			ScaleX:   playerSpec.Transform.ScaleX,
			ScaleY:   playerSpec.Transform.ScaleY,
			Rotation: playerSpec.Transform.Rotation,
		},
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

	spriteSheet, err := assets.LoadImage(playerSpec.Sprite.Image)
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

	return entity, nil
}
