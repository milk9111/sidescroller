package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewCamera(w *ecs.World) (ecs.Entity, error) {
	return BuildEntity(w, "camera.yaml")
}

func NewCameraAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	camera, err := BuildEntity(w, "camera.yaml")
	if err != nil {
		return 0, err
	}
	if err := SetEntityTransform(w, camera, x, y, 0); err != nil {
		return 0, fmt.Errorf("camera: override transform: %w", err)
	}
	return camera, nil
}
