package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type ColorSystem struct{}

func NewColorSystem() *ColorSystem { return &ColorSystem{} }

func (s *ColorSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach2(w, component.ColorComponent.Kind(), component.SpriteComponent.Kind(), func(_ ecs.Entity, c *component.Color, _ *component.Sprite) {
		if c == nil {
			return
		}
		c.R = clampColor01(c.R)
		c.G = clampColor01(c.G)
		c.B = clampColor01(c.B)
		c.A = clampColor01(c.A)
	})
}

func clampColor01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
