package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewAimTarget(w *ecs.World) (ecs.Entity, error) {
	aimTargetSpec, err := prefabs.LoadAimTargetSpec()
	if err != nil {
		return 0, fmt.Errorf("aim target: load spec: %w", err)
	}

	entity := w.CreateEntity()

	if err := ecs.Add(w, entity, component.AimTargetTagComponent, component.AimTargetTag{}); err != nil {
		return 0, fmt.Errorf("aim target: add tag: %w", err)
	}

	if err := ecs.Add(w, entity, component.TransformComponent, component.Transform{ScaleX: aimTargetSpec.Transform.ScaleX, ScaleY: aimTargetSpec.Transform.ScaleY}); err != nil {
		return 0, fmt.Errorf("aim target: add transform: %w", err)
	}

	if err := ecs.Add(w, entity, component.SpriteComponent, component.Sprite{
		Image:   nil,
		OriginX: aimTargetSpec.Sprite.OriginX,
		OriginY: aimTargetSpec.Sprite.OriginY,
	}); err != nil {
		return 0, fmt.Errorf("aim target: add sprite: %w", err)
	}

	if err := ecs.Add(w, entity, component.RenderLayerComponent, component.RenderLayer{Index: aimTargetSpec.RenderLayer.Index}); err != nil {
		return 0, fmt.Errorf("aim target: add render layer: %w", err)
	}

	if err := ecs.Add(w, entity, component.LineRenderComponent, component.LineRender{
		Width:     aimTargetSpec.LineRender.Width,
		Color:     aimTargetSpec.LineRender.Color.Color,
		AntiAlias: aimTargetSpec.LineRender.AntiAlias,
	}); err != nil {
		return 0, fmt.Errorf("aim target: add line render: %w", err)
	}

	return entity, nil
}
