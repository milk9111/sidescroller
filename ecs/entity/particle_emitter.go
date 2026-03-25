package entity

import "github.com/milk9111/sidescroller/ecs"

func NewParticleEmitter(w *ecs.World) (ecs.Entity, error) {
	return BuildEntity(w, "emitter_test.yaml")
}
