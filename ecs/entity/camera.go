package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewCamera(w *ecs.World) (ecs.Entity, error) {
	cameraSpec, err := prefabs.LoadCameraSpec()
	if err != nil {
		return 0, fmt.Errorf("camera: load spec: %w", err)
	}

	camera := ecs.CreateEntity(w)
	if err := ecs.Add(w, camera, component.CameraTagComponent.Kind(), &component.CameraTag{}); err != nil {
		return 0, fmt.Errorf("camera: add camera tag: %w", err)
	}

	if err := ecs.Add(w, camera, component.TransformComponent.Kind(), &component.Transform{
		X:        cameraSpec.Transform.X,
		Y:        cameraSpec.Transform.Y,
		ScaleX:   cameraSpec.Transform.ScaleX,
		ScaleY:   cameraSpec.Transform.ScaleY,
		Rotation: cameraSpec.Transform.Rotation,
	}); err != nil {
		return 0, fmt.Errorf("camera: add transform: %w", err)
	}

	smooth := cameraSpec.Smoothness
	if smooth == 0 {
		smooth = 0.15
	}
	// Use look config from spec if present; provide sensible defaults.
	lookOffset := cameraSpec.LookOffset
	if lookOffset == 0 {
		lookOffset = 48.0
	}
	lookSmooth := cameraSpec.LookSmooth
	if lookSmooth == 0 {
		lookSmooth = 0.15
	}
	if err := ecs.Add(w, camera, component.CameraComponent.Kind(), &component.Camera{
		TargetName: cameraSpec.Target,
		Zoom:       cameraSpec.Zoom,
		Smoothness: smooth,
		LookOffset: lookOffset,
		LookSmooth: lookSmooth,
	}); err != nil {
		return 0, fmt.Errorf("camera: add camera component: %w", err)
	}

	return camera, nil
}

func NewCameraAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	camera, err := NewCamera(w)
	if err != nil {
		return 0, err
	}
	transform, ok := ecs.Get(w, camera, component.TransformComponent.Kind())
	if !ok {
		transform = &component.Transform{ScaleX: 1, ScaleY: 1}
	}
	transform.X = x
	transform.Y = y
	if err := ecs.Add(w, camera, component.TransformComponent.Kind(), transform); err != nil {
		return 0, fmt.Errorf("camera: override transform: %w", err)
	}
	return camera, nil
}
