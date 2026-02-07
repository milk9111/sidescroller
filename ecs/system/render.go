package system

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type RenderSystem struct {
	camEntity ecs.Entity
}

func NewRenderSystem() *RenderSystem {
	return &RenderSystem{}
}

func (r *RenderSystem) Draw(w *ecs.World, screen *ebiten.Image) {
	if r == nil {
		return
	}

	if !r.camEntity.Valid() {
		if camEntity, ok := w.First(component.CameraComponent.Kind()); ok {
			r.camEntity = camEntity
		}
	}

	camX, camY := 0.0, 0.0
	// Fetch the camera entity's transform
	if camTransform, ok := ecs.Get(w, r.camEntity, component.TransformComponent); ok {
		camX = camTransform.X
		camY = camTransform.Y
	}

	for _, e := range w.Query(component.TransformComponent.Kind(), component.SpriteComponent.Kind()) {
		t, ok := ecs.Get(w, e, component.TransformComponent)
		if !ok {
			continue
		}

		s, ok := ecs.Get(w, e, component.SpriteComponent)
		if !ok || s.Image == nil {
			continue
		}

		img := s.Image
		if s.UseSource {
			sub, ok := s.Image.SubImage(s.Source).(*ebiten.Image)
			if ok {
				img = sub
			}
		}

		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-s.OriginX, -s.OriginY)

		sx := t.ScaleX
		if sx == 0 {
			sx = 1
		}

		sy := t.ScaleY
		if sy == 0 {
			sy = 1
		}

		op.GeoM.Scale(sx, sy)
		op.GeoM.Rotate(t.Rotation)
		op.GeoM.Translate(t.X-camX, t.Y-camY)

		screen.DrawImage(img, op)
	}
}
