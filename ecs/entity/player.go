package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewPlayer(w *ecs.World) (ecs.Entity, error) {
	return BuildEntity(w, "player.yaml")
}

func NewPlayerAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	entity, err := BuildEntity(w, "player.yaml")
	if err != nil {
		return 0, err
	}
	if err := SetEntityTransform(w, entity, x, y, 0); err != nil {
		return 0, fmt.Errorf("player: override transform: %w", err)
	}
	return entity, nil
}
