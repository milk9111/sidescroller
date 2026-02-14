package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

// AINavigationSystem computes whether there is ground slightly ahead of each
// AI entity on both left and right directions and stores the result in the
// AINavigation component so AI actions can consult it cheaply.
type AINavigationSystem struct{}

func NewAINavigationSystem() *AINavigationSystem {
	return &AINavigationSystem{}
}

func (s *AINavigationSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	// Gather static colliders (level geometry) as rectangles to test points against.
	type rect struct{ x, y, w, h float64 }
	staticRects := make([]rect, 0, 128)
	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, b *component.PhysicsBody, t *component.Transform) {
		if b == nil || t == nil || !b.Static {
			return
		}
		ow := b.Width
		oh := b.Height
		if ow <= 0 || oh <= 0 {
			return
		}
		ox := t.X + b.OffsetX
		oy := t.Y + b.OffsetY
		if !b.AlignTopLeft {
			ox = t.X + b.OffsetX - ow/2
			oy = t.Y + b.OffsetY - oh/2
		}
		staticRects = append(staticRects, rect{x: ox, y: oy, w: ow, h: oh})
	})

	// For each AI, compute foot points ahead on left and right and test for coverage.
	ecs.ForEach3(w, component.AINavigationComponent.Kind(), component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, nav *component.AINavigation, b *component.PhysicsBody, t *component.Transform) {
		if b == nil || t == nil || nav == nil {
			return
		}
		width := b.Width
		height := b.Height
		if width <= 0 {
			width = 32
		}
		if height <= 0 {
			height = 32
		}
		topLeftX := t.X + b.OffsetX
		topLeftY := t.Y + b.OffsetY
		if !b.AlignTopLeft {
			topLeftX = t.X + b.OffsetX - width/2
			topLeftY = t.Y + b.OffsetY - height/2
		}

		footY := topLeftY + height + 2.0
		footRightX := topLeftX + width + 1.0
		footLeftX := topLeftX - 1.0

		foundRight := false
		foundLeft := false
		for _, r := range staticRects {
			if !foundRight {
				if footRightX >= r.x && footRightX <= r.x+r.w && footY >= r.y && footY <= r.y+r.h {
					foundRight = true
				}
			}
			if !foundLeft {
				if footLeftX >= r.x && footLeftX <= r.x+r.w && footY >= r.y && footY <= r.y+r.h {
					foundLeft = true
				}
			}
			if foundLeft && foundRight {
				break
			}
		}

		nav.GroundAheadRight = foundRight
		nav.GroundAheadLeft = foundLeft
	})
}
