package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func NewAimTarget(w *ecs.World) (ecs.Entity, error) {
	entity := w.CreateEntity()

	if err := ecs.Add(w, entity, component.AimTargetTagComponent, component.AimTargetTag{}); err != nil {
		return 0, fmt.Errorf("aim target: add tag: %w", err)
	}

	img, err := assets.LoadImage("aim_target.png")
	if err != nil {
		return 0, fmt.Errorf("aim target: load sprite: %w", err)
	}

	originX := float64(img.Bounds().Dx()) / 2
	originY := float64(img.Bounds().Dy()) / 2

	if err := ecs.Add(w, entity, component.TransformComponent, component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		return 0, fmt.Errorf("aim target: add transform: %w", err)
	}

	if err := ecs.Add(w, entity, component.SpriteComponent, component.Sprite{
		Image:   nil,
		OriginX: originX,
		OriginY: originY,
	}); err != nil {
		return 0, fmt.Errorf("aim target: add sprite: %w", err)
	}

	if err := ecs.Add(w, entity, component.RenderLayerComponent, component.RenderLayer{Index: 1000}); err != nil {
		return 0, fmt.Errorf("aim target: add render layer: %w", err)
	}

	return entity, nil
}
