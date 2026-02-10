package entity

import (
	"fmt"
	"image/color"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewAnchor(w *ecs.World) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("anchor: world is nil")
	}

	anchorSpec, err := prefabs.LoadAnchorSpec()
	if err != nil {
		return 0, fmt.Errorf("anchor: load spec: %w", err)
	}

	entity := w.CreateEntity()

	if err := ecs.Add(w, entity, component.TransformComponent, component.Transform{
		ScaleX: anchorSpec.Transform.ScaleX,
		ScaleY: anchorSpec.Transform.ScaleY,
	}); err != nil {
		return 0, fmt.Errorf("anchor: add transform: %w", err)
	}

	img, err := assets.LoadImage(anchorSpec.Sprite.Image)
	if err != nil {
		return 0, fmt.Errorf("anchor: load sprite image: %w", err)
	}

	originX := anchorSpec.Sprite.OriginX
	originY := anchorSpec.Sprite.OriginY
	if originX == 0 && originY == 0 {
		originX = float64(img.Bounds().Dx()) / 2
		originY = float64(img.Bounds().Dy()) / 2
	}

	if err := ecs.Add(w, entity, component.AnchorTagComponent, component.AnchorTag{}); err != nil {
		return 0, fmt.Errorf("anchor: add tag: %w", err)
	}

	if err := ecs.Add(w, entity, component.SpriteComponent, component.Sprite{
		Image:   img,
		OriginX: originX,
		OriginY: originY,
	}); err != nil {
		return 0, fmt.Errorf("anchor: add sprite: %w", err)
	}

	if err := ecs.Add(w, entity, component.LineRenderComponent, component.LineRender{
		Width:     2,
		Color:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
		AntiAlias: false,
	}); err != nil {
		return 0, fmt.Errorf("anchor: add line render: %w", err)
	}

	// Anchor is kinematic (no physics body); movement and attachment
	// are handled by AnchorSystem which will create joints against the world.

	if err := ecs.Add(w, entity, component.RenderLayerComponent, component.RenderLayer{Index: anchorSpec.RenderLayer.Index}); err != nil {
		return 0, fmt.Errorf("anchor: add render layer: %w", err)
	}

	if err := ecs.Add(w, entity, component.AnchorComponent, component.Anchor{
		Speed: anchorSpec.Speed,
	}); err != nil {
		return 0, fmt.Errorf("anchor: add anchor component: %w", err)
	}

	return entity, nil
}

func NewAnchorAt(w *ecs.World, x, y, rotation float64) (ecs.Entity, error) {
	anchor, err := NewAnchor(w)
	if err != nil {
		return 0, err
	}

	transform, ok := ecs.Get(w, anchor, component.TransformComponent)
	if !ok {
		return 0, fmt.Errorf("anchor: missing transform component")
	}

	transform.X = x
	transform.Y = y
	transform.Rotation = rotation
	if err := ecs.Add(w, anchor, component.TransformComponent, transform); err != nil {
		return 0, fmt.Errorf("anchor: override transform: %w", err)
	}

	return anchor, nil
}
