package entity

import "github.com/milk9111/sidescroller/ecs"

func NewTransition(world *ecs.World) (ecs.Entity, error) {
	return BuildEntity(world, "transition.yaml")
}
