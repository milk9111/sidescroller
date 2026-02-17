package system

import (
	"math"
	"math/rand"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type ClusterRepulsionSystem struct {
	Radius   float64
	Strength float64
}

func NewClusterRepulsionSystem() *ClusterRepulsionSystem {
	return &ClusterRepulsionSystem{
		Radius:   24.0,
		Strength: 40.0,
	}
}

func (cr *ClusterRepulsionSystem) Update(w *ecs.World) {
	if cr == nil || w == nil {
		return
	}

	// Nested iteration over entities with the needed components. Use two ForEach3
	// calls and only process pairs once (uint64(e1) < uint64(e2)).
	ecs.ForEach3(w, component.AITagComponent.Kind(), component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e1 ecs.Entity, _ *component.AITag, b1 *component.PhysicsBody, t1 *component.Transform) {
		if b1 == nil || b1.Body == nil || b1.Static {
			return
		}

		ecs.ForEach3(w, component.AITagComponent.Kind(), component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e2 ecs.Entity, _ *component.AITag, b2 *component.PhysicsBody, t2 *component.Transform) {
			if b2 == nil || b2.Body == nil || b2.Static {
				return
			}
			// only handle each unordered pair once and skip self
			if uint64(e1) >= uint64(e2) {
				return
			}

			// compute vector from e2 -> e1 (push apart)
			dx := b1.Body.Position().X - b2.Body.Position().X
			dy := b1.Body.Position().Y - b2.Body.Position().Y
			dist := math.Hypot(dx, dy)
			if dist == 0 {
				dx = (rand.Float64() - 0.5) * 1e-3
				dy = (rand.Float64() - 0.5) * 1e-3
				dist = math.Hypot(dx, dy)
			}

			if dist >= cr.Radius {
				return
			}

			// normalized direction
			nx := dx / dist
			ny := dy / dist

			// strength falloff: linear with overlap
			overlap := cr.Radius - dist
			mag := cr.Strength * (overlap / cr.Radius)

			// apply half impulse to each body to separate
			ix := nx * mag * 0.5
			iy := ny * mag * 0.5

			if b1 != nil && b1.Body != nil {
				b1.Body.ApplyImpulseAtWorldPoint(cp.Vector{X: ix, Y: iy}, b1.Body.Position())
			}
			if b2 != nil && b2.Body != nil {
				b2.Body.ApplyImpulseAtWorldPoint(cp.Vector{X: -ix, Y: -iy}, b2.Body.Position())
			}
		})
	})
}
