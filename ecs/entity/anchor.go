package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewAnchor(w *ecs.World) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("anchor: world is nil")
	}
	return BuildEntity(w, "anchor.yaml")
}

func NewAnchorAt(w *ecs.World, x, y, rotation float64) (ecs.Entity, error) {
	anchor, err := BuildEntity(w, "anchor.yaml")
	if err != nil {
		return 0, err
	}
	if err := SetEntityTransform(w, anchor, x, y, rotation); err != nil {
		return 0, fmt.Errorf("anchor: override transform: %w", err)
	}
	return anchor, nil
}
