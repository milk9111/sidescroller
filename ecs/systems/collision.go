package systems

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// CollisionSystem steps physics and updates collision state.
type CollisionSystem struct {
	StepScale float64
}

// NewCollisionSystem creates a CollisionSystem.
func NewCollisionSystem() *CollisionSystem {
	return &CollisionSystem{StepScale: 1.0}
}

// Update steps physics and syncs transforms/velocities.
func (s *CollisionSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	pw := w.PhysicsWorld()
	if pw == nil {
		return
	}
	trs := w.Transforms()
	cols := w.Colliders()
	bodies := w.PhysicsBodies()
	states := w.CollisionStates()
	grounders := w.GroundSensors()
	vels := w.Velocities()
	gravs := w.Gravities()

	if trs == nil || cols == nil || bodies == nil {
		return
	}

	for _, id := range cols.Entities() {
		cv := cols.Get(id)
		col, ok := cv.(*components.Collider)
		if !ok || col == nil || col.Static {
			continue
		}
		tv := trs.Get(id)
		tr, ok := tv.(*components.Transform)
		if !ok || tr == nil {
			continue
		}

		var gs *components.GroundSensor
		if grounders != nil {
			if gv := grounders.Get(id); gv != nil {
				gs, _ = gv.(*components.GroundSensor)
			}
		}
		var grav *components.Gravity
		if gravs != nil {
			if gv := gravs.Get(id); gv != nil {
				grav, _ = gv.(*components.Gravity)
			}
		}

		var body *components.PhysicsBody
		if bv := bodies.Get(id); bv != nil {
			body, _ = bv.(*components.PhysicsBody)
		}
		body = pw.EnsureBody(id, tr, col, gs, grav, body)
		if body != nil {
			bodies.Set(id, body)
		}

		var state *components.CollisionState
		if states != nil {
			if sv := states.Get(id); sv != nil {
				state, _ = sv.(*components.CollisionState)
			}
			if state == nil {
				state = &components.CollisionState{}
				states.Set(id, state)
			}
			if state.GroundGrace > 0 {
				state.GroundGrace--
			}
			state.Grounded = false
			state.Wall = components.WallNone
			state.HitHazard = false
			pw.SetEntityState(id, state)
		}
	}

	if s.StepScale <= 0 {
		s.StepScale = 1.0
	}
	pw.Step(s.StepScale)

	for _, id := range bodies.Entities() {
		bv := bodies.Get(id)
		body, ok := bv.(*components.PhysicsBody)
		if !ok || body == nil || body.Body == nil {
			continue
		}
		cv := cols.Get(id)
		col, ok := cv.(*components.Collider)
		if !ok || col == nil {
			continue
		}
		tv := trs.Get(id)
		tr, ok := tv.(*components.Transform)
		if !ok || tr == nil {
			continue
		}
		pos := body.Body.Position()
		tr.X = float32(pos.X - float64(col.Width)/2.0 - float64(col.OffsetX))
		tr.Y = float32(pos.Y - float64(col.Height)/2.0 - float64(col.OffsetY))

		if vels != nil {
			if vv := vels.Get(id); vv != nil {
				if vel, ok := vv.(*components.Velocity); ok && vel != nil {
					v := body.Body.Velocity()
					vel.VX = float32(v.X)
					vel.VY = float32(v.Y)
				}
			}
		}

		if states != nil {
			if sv := states.Get(id); sv != nil {
				if state, ok := sv.(*components.CollisionState); ok && state != nil {
					if state.GroundGrace > 0 {
						state.Grounded = true
					}
					if state.Grounded && !state.PrevGrounded {
						w.Events().Push(ecs.Event{Type: "collision", Data: ecs.CollisionEvent{Entity: ecs.Entity{ID: id, Gen: 0}, Kind: ecs.CollisionEventGrounded}})
					}
					if state.HitHazard && !state.PrevHitHazard {
						w.Events().Push(ecs.Event{Type: "collision", Data: ecs.CollisionEvent{Entity: ecs.Entity{ID: id, Gen: 0}, Kind: ecs.CollisionEventHitHazard}})
					}
					state.PrevGrounded = state.Grounded
					state.PrevWall = state.Wall
					state.PrevHitHazard = state.HitHazard
				}
			}
		}
	}
}
