package system

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type RenderSystem struct {
	Transform component.ComponentHandle[component.Transform]
	Sprite    component.ComponentHandle[component.Sprite]
}

func NewRenderSystem(transform component.ComponentHandle[component.Transform], sprite component.ComponentHandle[component.Sprite]) *RenderSystem {
	return &RenderSystem{Transform: transform, Sprite: sprite}
}

func (r *RenderSystem) Draw(w *ecs.World, screen *ebiten.Image) {
	if r == nil {
		return
	}
	for _, e := range w.Query(r.Transform.Kind(), r.Sprite.Kind()) {
		t, ok := ecs.Get(w, e, r.Transform)
		if !ok {
			continue
		}
		s, ok := ecs.Get(w, e, r.Sprite)
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
		op.GeoM.Translate(t.X, t.Y)
		screen.DrawImage(img, op)
	}
}
