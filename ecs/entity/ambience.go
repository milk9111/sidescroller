package entity

import "github.com/milk9111/sidescroller/ecs"

func NewAmbience(w *ecs.World) (ecs.Entity, error) {
	return BuildEntity(w, "ambience.yaml")
}
