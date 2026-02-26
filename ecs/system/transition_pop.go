package system

import (
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

// TransitionPopSystem processes `TransitionPop` components, applies the launch
// once, and keeps the component present until the player is grounded again.
type TransitionPopSystem struct{}

func NewTransitionPopSystem() *TransitionPopSystem { return &TransitionPopSystem{} }

func (s *TransitionPopSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach2(w, component.TransitionPopComponent.Kind(), component.PhysicsBodyComponent.Kind(), func(e ecs.Entity, pop *component.TransitionPop, bodyComp *component.PhysicsBody) {
		if bodyComp == nil || bodyComp.Body == nil {
			return
		}

		if !pop.Applied {
			bodyComp.Body.SetVelocityVector(cp.Vector{X: pop.VX, Y: pop.VY})

			if spriteComp, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && spriteComp != nil {
				spriteComp.FacingLeft = pop.FacingLeft
				_ = ecs.Add(w, e, component.SpriteComponent.Kind(), spriteComp)
			}

			if stateComp, ok := ecs.Get(w, e, component.PlayerStateMachineComponent.Kind()); ok && stateComp != nil {
				if pop.WallJumpDur > 0 {
					stateComp.WallJumpTimer = pop.WallJumpDur
					stateComp.WallJumpX = pop.WallJumpX
					_ = ecs.Add(w, e, component.PlayerStateMachineComponent.Kind(), stateComp)
				}
			}

			pop.Applied = true
			_ = ecs.Add(w, e, component.TransitionPopComponent.Kind(), pop)
			return
		}

		grounded := false
		if pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent.Kind()); ok && pc != nil {
			grounded = pc.Grounded || pc.GroundGrace > 0
		}

		if !grounded {
			if !pop.Airborne {
				pop.Airborne = true
				_ = ecs.Add(w, e, component.TransitionPopComponent.Kind(), pop)
			}
			return
		}

		if pop.Airborne {
			_ = ecs.Remove(w, e, component.TransitionPopComponent.Kind())
		}
	})
}
