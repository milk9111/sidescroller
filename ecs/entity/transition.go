package entity

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewTransition(world *ecs.World) (ecs.Entity, error) {
	spec, err := prefabs.LoadSpec[prefabs.TransitionSpec]("transition.yaml")
	if err != nil {
		return 0, fmt.Errorf("transition: failed to load transition spec: %w", err)
	}

	entity := world.CreateEntity()

	if err := ecs.Add(world, entity, component.TransformComponent, component.Transform{
		X:        spec.Transform.X,
		Y:        spec.Transform.Y,
		ScaleX:   spec.Transform.ScaleX,
		ScaleY:   spec.Transform.ScaleY,
		Rotation: spec.Transform.Rotation,
	}); err != nil {
		return 0, fmt.Errorf("transition: failed to add transform component: %w", err)
	}

	var img *ebiten.Image
	if spec.Sprite.Image != "" {
		img, err = assets.LoadImage(spec.Sprite.Image)
		if err != nil {
			return 0, fmt.Errorf("transition: failed to load sprite image: %w", err)
		}
	}

	if err := ecs.Add(world, entity, component.SpriteComponent, component.Sprite{
		Image:     img,
		UseSource: spec.Sprite.UseSource,
		OriginX:   spec.Sprite.OriginX,
		OriginY:   spec.Sprite.OriginY,
	}); err != nil {
		return 0, fmt.Errorf("transition: failed to add sprite component: %w", err)
	}

	if err := ecs.Add(world, entity, component.RenderLayerComponent, component.RenderLayer{
		Index: spec.RenderLayer.Index,
	}); err != nil {
		return 0, fmt.Errorf("transition: failed to add render layer component: %w", err)
	}

	spriteSheet, err := assets.LoadImage(spec.Animation.Sheet)
	if err != nil {
		return 0, fmt.Errorf("transition: failed to load sprite sheet: %w", err)
	}

	defs := make(map[string]component.AnimationDef, len(spec.Animation.Defs))
	for name, defSpec := range spec.Animation.Defs {
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
		world,
		entity,
		component.AnimationComponent,
		component.Animation{
			Sheet:      spriteSheet,
			Defs:       defs,
			Current:    spec.Animation.Current,
			Frame:      0,
			FrameTimer: 0,
			Playing:    true,
		},
	); err != nil {
		return 0, fmt.Errorf("transition: failed to add animation component: %w", err)
	}

	return entity, nil
}
