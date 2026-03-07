package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type ParallaxSystem struct{}

func NewParallaxSystem() *ParallaxSystem { return &ParallaxSystem{} }

func (s *ParallaxSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	camEntity, ok := ecs.First(w, component.CameraComponent.Kind())
	if !ok {
		return
	}

	camTransform, ok := ecs.Get(w, camEntity, component.TransformComponent.Kind())
	if !ok || camTransform == nil {
		return
	}

	camX, camY := cameraPosition(camTransform)

	ecs.ForEach2(w, component.ParallaxComponent.Kind(), component.TransformComponent.Kind(), func(_ ecs.Entity, parallax *component.Parallax, t *component.Transform) {
		if parallax == nil || t == nil {
			return
		}

		if !parallax.Initialized {
			parallax.BaseX = t.X
			parallax.BaseY = t.Y
			parallax.CameraBaseX = camX
			parallax.CameraBaseY = camY
			if parallax.HasAnchorCameraX {
				parallax.CameraBaseX = parallax.AnchorCameraX
			}
			if parallax.HasAnchorCameraY {
				parallax.CameraBaseY = parallax.AnchorCameraY
			}
			parallax.Initialized = true
		}

		dx := camX - parallax.CameraBaseX
		dy := camY - parallax.CameraBaseY
		t.X = parallax.BaseX + dx*parallax.FactorX
		t.Y = parallax.BaseY + dy*parallax.FactorY
	})
}

func cameraPosition(t *component.Transform) (float64, float64) {
	if t == nil {
		return 0, 0
	}
	if t.Parent != 0 {
		return t.WorldX, t.WorldY
	}
	return t.X, t.Y
}
