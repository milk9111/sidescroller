package systems

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/obj"
)

// CameraSystem updates the legacy camera from ECS data.
type CameraSystem struct {
	Camera       *obj.Camera
	CameraEntity ecs.Entity
}

// NewCameraSystem creates a CameraSystem.
func NewCameraSystem(camera *obj.Camera, cameraEntity ecs.Entity) *CameraSystem {
	return &CameraSystem{Camera: camera, CameraEntity: cameraEntity}
}

// Update moves the camera toward its target.
func (s *CameraSystem) Update(w *ecs.World) {
	if w == nil || s == nil || s.Camera == nil || s.CameraEntity.ID == 0 {
		return
	}
	follow := w.GetCameraFollow(s.CameraEntity)
	if follow == nil || follow.TargetEntity == 0 {
		return
	}
	tv := w.Transforms().Get(follow.TargetEntity)
	tr, ok := tv.(*components.Transform)
	if !ok || tr == nil {
		return
	}

	width := float32(0)
	height := float32(0)
	if cv := w.Colliders().Get(follow.TargetEntity); cv != nil {
		if col, ok := cv.(*components.Collider); ok && col != nil {
			width = col.Width
			height = col.Height
		}
	}
	if width == 0 || height == 0 {
		if sv := w.Sprites().Get(follow.TargetEntity); sv != nil {
			if spr, ok := sv.(*components.Sprite); ok && spr != nil {
				width = spr.Width
				height = spr.Height
			}
		}
	}

	cx := float64(tr.X) + float64(width)/2.0 + follow.OffsetX
	cy := float64(tr.Y) + float64(height)/2.0 + follow.OffsetY
	s.Camera.Update(cx, cy)

	state := w.GetCameraState(s.CameraEntity)
	if state == nil {
		state = &components.CameraState{}
		w.SetCameraState(s.CameraEntity, state)
	}
	state.PosX = s.Camera.PosX
	state.PosY = s.Camera.PosY
	state.Zoom = s.Camera.Zoom()
}
