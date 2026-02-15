package system

import (
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type RespawnSystem struct{}

func NewRespawnSystem() *RespawnSystem { return &RespawnSystem{} }

// Update performs pending respawn requests for players. It should run after
// the PhysicsSystem so any anchor constraints marked for removal have been
// processed.
func (s *RespawnSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach(w, component.RespawnRequestComponent.Kind(), func(e ecs.Entity, _ *component.RespawnRequest) {
		// Only handle player entity
		if !ecs.Has(w, e, component.PlayerTagComponent.Kind()) {
			// remove the request so it doesn't linger
			_ = ecs.Remove(w, e, component.RespawnRequestComponent.Kind())
			return
		}

		t, tok := ecs.Get(w, e, component.TransformComponent.Kind())
		if !tok || t == nil {
			_ = ecs.Remove(w, e, component.RespawnRequestComponent.Kind())
			return
		}

		safe, sok := ecs.Get(w, e, component.SafeRespawnComponent.Kind())
		if sok && safe != nil && safe.Initialized {
			t.X = safe.X
			t.Y = safe.Y
			_ = ecs.Add(w, e, component.TransformComponent.Kind(), t)

			if body, bok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); bok && body != nil && body.Body != nil {
				centerX := t.X + body.OffsetX
				centerY := t.Y + body.OffsetY
				if body.AlignTopLeft {
					centerX += body.Width / 2
					centerY += body.Height / 2
				}
				body.Body.SetPosition(cp.Vector{X: centerX, Y: centerY})
				body.Body.SetVelocityVector(cp.Vector{})
				body.Body.SetAngularVelocity(0)
				_ = ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), body)
			}
		}

		_ = ecs.Remove(w, e, component.RespawnRequestComponent.Kind())
	})
}
