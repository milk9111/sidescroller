package entity

import "github.com/milk9111/sidescroller/ecs"

func NewAimTarget(w *ecs.World) (ecs.Entity, error) {
	return BuildEntity(w, "aim_target.yaml")
}
