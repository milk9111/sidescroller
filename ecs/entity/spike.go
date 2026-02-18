package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func NewSpike(w *ecs.World) (ecs.Entity, error) {
	return BuildEntity(w, "spike.yaml")
}

func NewSpikeAt(w *ecs.World, x, y, rotation float64) (ecs.Entity, error) {
	e, err := BuildEntity(w, "spike.yaml")
	if err != nil {
		return 0, err
	}
	t, _ := ecs.Get(w, e, component.TransformComponent.Kind())
	if t == nil {
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
