package ecs

import "github.com/hajimehoshi/ebiten/v2"

// RenderSystem draws ECS entities each frame.
type RenderSystem interface {
	Draw(w *World, screen *ebiten.Image, camX, camY, zoom float64)
}

// Draw calls all render-capable systems.
func (w *World) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if w == nil || screen == nil {
		return
	}
	for _, s := range w.systems {
		rs, ok := s.(RenderSystem)
		if !ok || rs == nil {
			continue
		}
		rs.Draw(w, screen, camX, camY, zoom)
	}
}
