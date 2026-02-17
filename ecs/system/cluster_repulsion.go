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

	type entInfo struct {
		e    ecs.Entity
		body *component.PhysicsBody
		tr   *component.Transform
	}

	list := make([]entInfo, 0)

	ecs.ForEach3(w, component.AITagComponent.Kind(), component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, _ *component.AITag, body *component.PhysicsBody, tr *component.Transform) {
		if body == nil || body.Body == nil || body.Static {
			return
		}
		list = append(list, entInfo{e: e, body: body, tr: tr})
	})

	n := len(list)
	if n < 2 {
		return
	}

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			bi := list[i]
			bj := list[j]

			// compute vector from j -> i (push apart)
			dx := bi.body.Body.Position().X - bj.body.Body.Position().X
			dy := bi.body.Body.Position().Y - bj.body.Body.Position().Y
			dist := math.Hypot(dx, dy)
			if dist == 0 {
				dx = (rand.Float64() - 0.5) * 1e-3
				dy = (rand.Float64() - 0.5) * 1e-3
				dist = math.Hypot(dx, dy)
			}

			if dist >= cr.Radius {
				continue
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

			if bi.body != nil && bi.body.Body != nil {
				bi.body.Body.ApplyImpulseAtWorldPoint(cp.Vector{X: ix, Y: iy}, bi.body.Body.Position())
			}
			if bj.body != nil && bj.body.Body != nil {
				bj.body.Body.ApplyImpulseAtWorldPoint(cp.Vector{X: -ix, Y: -iy}, bj.body.Body.Position())
			}
		}
	}
}
