package entity

import (
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewPlayer(w *ecs.World) ecs.Entity {
	playerSpec, err := prefabs.LoadPlayerSpec()
	if err != nil {
		panic("failed to load player spec: " + err.Error())
	}

	entity := w.CreateEntity()
	ecs.Add(
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
	)

	spriteSheet, err := assets.LoadImage(playerSpec.Sprite.Image)
	if err != nil {
		panic("failed to load player sprite: " + err.Error())
	}

	ecs.Add(
		w,
		entity,
		component.SpriteComponent,
		component.Sprite{
			Image:     spriteSheet,
			UseSource: playerSpec.Sprite.UseSource,
			OriginX:   playerSpec.Sprite.OriginX,
			OriginY:   playerSpec.Sprite.OriginY,
		},
	)

	return entity
}
