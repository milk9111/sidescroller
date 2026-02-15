package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewSpike(w *ecs.World) (ecs.Entity, error) {
	// Try to load prefab spec for spikes; fall back to defaults on error
	var spec prefabs.SpikeSpec
	if s, err := prefabs.LoadSpec[prefabs.SpikeSpec]("spike.yaml"); err == nil {
		spec = s
	}

	imgName := spec.Sprite.Image
	if imgName == "" {
		imgName = "spikes.png"
	}
	img, err := assets.LoadImage(imgName)
	if err != nil {
		return 0, fmt.Errorf("spike: load image %s: %w", imgName, err)
	}

	e := ecs.CreateEntity(w)

	// Transform (use spec defaults if provided)
	tr := &component.Transform{
		ScaleX: 1,
		ScaleY: 1,
	}
	if spec.Transform.ScaleX != 0 {
		tr.ScaleX = spec.Transform.ScaleX
	}
	if spec.Transform.ScaleY != 0 {
		tr.ScaleY = spec.Transform.ScaleY
	}
	if err := ecs.Add(w, e, component.TransformComponent.Kind(), tr); err != nil {
		return 0, fmt.Errorf("spike: add transform: %w", err)
	}

	// Sprite
	sprite := &component.Sprite{
		Image:     img,
		UseSource: spec.Sprite.UseSource,
		OriginX:   spec.Sprite.OriginX,
		OriginY:   spec.Sprite.OriginY,
	}
	// If the prefab didn't specify an origin, default to image center so
	// rotations don't visually displace the sprite relative to its collider.
	if sprite.OriginX == 0 && sprite.OriginY == 0 {
		iw, ih := img.Size()
		sprite.OriginX = float64(iw) / 2.0
		sprite.OriginY = float64(ih) / 2.0
	}
	if err := ecs.Add(w, e, component.SpriteComponent.Kind(), sprite); err != nil {
		return 0, fmt.Errorf("spike: add sprite: %w", err)
	}

	// Render layer (spec or sensible default)
	rl := 95
	if spec.RenderLayer.Index != 0 {
		rl = spec.RenderLayer.Index
	}
	if err := ecs.Add(w, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: rl}); err != nil {
		return 0, fmt.Errorf("spike: add render layer: %w", err)
	}

	// Hazard/collider (prefer spec collider if provided)
	wid, hgt := img.Size()
	hazardW := float64(wid)
	hazardH := float64(hgt)
	offsetX := 0.0
	offsetY := 0.0
	if spec.Collider.Width != 0 {
		hazardW = spec.Collider.Width
	}
	if spec.Collider.Height != 0 {
		hazardH = spec.Collider.Height
	}
	if spec.Collider.OffsetX != 0 {
		offsetX = spec.Collider.OffsetX
	}
	if spec.Collider.OffsetY != 0 {
		offsetY = spec.Collider.OffsetY
	}
	if hazardW <= 0 {
		hazardW = 32
	}
	if hazardH <= 0 {
		hazardH = 32
	}
	if err := ecs.Add(w, e, component.HazardComponent.Kind(), &component.Hazard{
		Width:   hazardW,
		Height:  hazardH,
		OffsetX: offsetX,
		OffsetY: offsetY,
	}); err != nil {
		return 0, fmt.Errorf("spike: add hazard: %w", err)
	}

	return e, nil
}

func NewSpikeAt(w *ecs.World, x, y, rotation float64) (ecs.Entity, error) {
	e, err := NewSpike(w)
	if err != nil {
		return 0, err
	}
	t, ok := ecs.Get(w, e, component.TransformComponent.Kind())
	if !ok || t == nil {
		t = &component.Transform{ScaleX: 1, ScaleY: 1}
	}
	// Position the transform so that the sprite's origin aligns with the
	// provided (x,y) top-left coordinates from level data. If the sprite's
	// origin is centered, this moves the transform to the sprite center so
	// rotations keep the sprite visually aligned with the collider.
	if s, sok := ecs.Get(w, e, component.SpriteComponent.Kind()); sok && s != nil {
		t.X = x + s.OriginX
		t.Y = y + s.OriginY
	} else {
		t.X = x
		t.Y = y
	}
	t.Rotation = rotation
	if err := ecs.Add(w, e, component.TransformComponent.Kind(), t); err != nil {
		return 0, fmt.Errorf("spike: override transform: %w", err)
	}
	return e, nil
}
