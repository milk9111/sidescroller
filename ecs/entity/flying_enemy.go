package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewFlyingEnemy(w *ecs.World) (ecs.Entity, error) {
	return BuildEntity(w, "flying_enemy.yaml")
}

func NewFlyingEnemyAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	entity, err := BuildEntity(w, "flying_enemy.yaml")
	if err != nil {
		return 0, err
	}
	if err := SetEntityTransform(w, entity, x, y, 0); err != nil {
		return 0, fmt.Errorf("flying enemy: override transform: %w", err)
	}
	return entity, nil
}
