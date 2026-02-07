package systems

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// PlayerControllerSystem handles player movement/state from input.
type PlayerControllerSystem struct{}

// NewPlayerControllerSystem creates a PlayerControllerSystem.
func NewPlayerControllerSystem() *PlayerControllerSystem {
	return &PlayerControllerSystem{}
}

// Update applies movement, jumping, and state updates for players.
func (s *PlayerControllerSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	set := w.PlayerControllers()
	if set == nil {
		return
	}

	for _, id := range set.Entities() {
		pv := set.Get(id)
		ctrl, ok := pv.(*components.PlayerController)
		if !ok || ctrl == nil {
			continue
		}

		ent := ecs.Entity{ID: id, Gen: 0}
		input := w.GetInput(ent)
		vel := w.GetVelocity(ent)
		if vel == nil {
			vel = &components.Velocity{}
			w.SetVelocity(ent, vel)
		}
		colState := w.GetCollisionState(ent)
		grounded := false
		if colState != nil {
			grounded = colState.Grounded
		}

		if grounded {
			ctrl.CoyoteTimer = ctrl.CoyoteFrames
			ctrl.JumpsUsed = 0
		} else if ctrl.CoyoteTimer > 0 {
			ctrl.CoyoteTimer--
		}

		moveX := float32(0)
		if input != nil {
			moveX = input.MoveX
		}
		if moveX != 0 {
			ctrl.FacingRight = moveX > 0
		}

		desiredVX := moveX * ctrl.MoveSpeed
		if ctrl.MaxSpeedX > 0 {
			desiredVX = float32(math.Max(float64(-ctrl.MaxSpeedX), math.Min(float64(ctrl.MaxSpeedX), float64(desiredVX))))
		}
		vel.VX = desiredVX

		jumpPressed := false
		if input != nil {
			jumpPressed = input.JumpPressed
		}
		canJump := grounded || ctrl.CoyoteTimer > 0
		if !canJump && ctrl.DoubleJump && ctrl.MaxJumps > 0 {
			canJump = ctrl.JumpsUsed < ctrl.MaxJumps
		}
		if jumpPressed && canJump {
			vel.VY = ctrl.JumpVelocity
			ctrl.CoyoteTimer = 0
			ctrl.JumpsUsed++
		}

		if grounded {
			if moveX == 0 {
				ctrl.State = "idle"
			} else {
				ctrl.State = "run"
			}
		} else if vel.VY < 0 {
			ctrl.State = "jump"
		} else {
			ctrl.State = "fall"
		}

		if spr := w.GetSprite(ent); spr != nil {
			spr.FlipX = !ctrl.FacingRight
		}
		if anim := w.GetAnimator(ent); anim != nil {
			desired := ctrl.IdleAnim
			if ctrl.State == "run" {
				desired = ctrl.RunAnim
			}
			if desired != nil && anim.Anim != desired {
				anim.Anim = desired
				anim.Anim.Reset()
			}
		}
	}
}
