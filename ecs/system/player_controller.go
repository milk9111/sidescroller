package system

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	// small threshold to treat near-zero vertical velocity as grounded
	groundedEpsilon = 0.1
)

type PlayerControllerSystem struct{}

func NewPlayerControllerSystem() *PlayerControllerSystem {
	return &PlayerControllerSystem{}
}

func (p *PlayerControllerSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	entities := w.Query(
		component.PlayerTagComponent.Kind(),
		component.PlayerComponent.Kind(),
		component.InputComponent.Kind(),
		component.PhysicsBodyComponent.Kind(),
		component.PlayerStateMachineComponent.Kind(),
		component.AnimationComponent.Kind(),
		component.SpriteComponent.Kind(),
	)
	for _, e := range entities {
		input, ok := ecs.Get(w, e, component.InputComponent)
		if !ok {
			continue
		}

		player, ok := ecs.Get(w, e, component.PlayerComponent)
		if !ok {
			continue
		}

		bodyComp, ok := ecs.Get(w, e, component.PhysicsBodyComponent)
		if !ok || bodyComp.Body == nil {
			continue
		}

		animComp, ok := ecs.Get(w, e, component.AnimationComponent)
		if !ok {
			continue
		}

		spriteComp, ok := ecs.Get(w, e, component.SpriteComponent)
		if !ok {
			continue
		}

		stateComp, ok := ecs.Get(w, e, component.PlayerStateMachineComponent)
		if !ok {
			stateComp = component.PlayerStateMachine{}
		}

		// helper ground check used by multiple closures
		isGroundedFn := func() bool {
			// Prefer chipmunk-derived grounded state when available.
			if pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent); ok {
				if pc.Grounded || pc.GroundGrace > 0 {
					return true
				}
			}
			// fallback to AABB-based check if no collision component is present
			if bodyComp.Body == nil {
				return false
			}
			pos := bodyComp.Body.Position()
			halfW := bodyComp.Width / 2.0
			halfH := bodyComp.Height / 2.0
			playerBottom := pos.Y + halfH

			// how far below the bottom we consider "ground" (in world units)
			const groundCheckDist = 0.5

			for _, other := range w.Query(component.PhysicsBodyComponent.Kind()) {
				if other == e {
					continue
				}
				oComp, ok := ecs.Get(w, other, component.PhysicsBodyComponent)
				if !ok || oComp.Body == nil {
					continue
				}

				oPos := oComp.Body.Position()
				oHalfW := oComp.Width / 2.0
				oHalfH := oComp.Height / 2.0
				otherTop := oPos.Y - oHalfH

				// horizontal overlap check
				if math.Abs(pos.X-oPos.X) > (halfW + oHalfW) {
					continue
				}

				// is player's bottom within groundCheckDist of other's top?
				if playerBottom >= otherTop-groundCheckDist && playerBottom <= otherTop+groundCheckDist {
					return true
				}
			}
			return false
		}

		ctx := component.PlayerStateContext{
			Input:  &input,
			Player: &player,
			GetVelocity: func() (x, y float64) {
				vel := bodyComp.Body.Velocity()
				return vel.X, vel.Y
			},
			SetVelocity: func(x, y float64) {
				bodyComp.Body.SetVelocityVector(cp.Vector{X: x, Y: y})
			},
			SetAngle: func(angle float64) {
				bodyComp.Body.SetAngle(angle)
			},
			SetAngularVelocity: func(omega float64) {
				bodyComp.Body.SetAngularVelocity(omega)
			},
			IsGrounded: isGroundedFn,
			WallSide: func() int {
				if pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent); ok {
					return pc.Wall
				}
				return 0
			},
			GetWallGrabTimer: func() int {
				return stateComp.WallGrabTimer
			},
			SetWallGrabTimer: func(frames int) {
				stateComp.WallGrabTimer = frames
			},
			GetWallJumpTimer: func() int {
				return stateComp.WallJumpTimer
			},
			SetWallJumpTimer: func(frames int) {
				stateComp.WallJumpTimer = frames
			},
			GetJumpHoldTimer: func() int {
				return stateComp.JumpHoldTimer
			},
			SetJumpHoldTimer: func(frames int) {
				stateComp.JumpHoldTimer = frames
			},
			GetWallJumpX: func() float64 {
				return stateComp.WallJumpX
			},
			SetWallJumpX: func(x float64) {
				stateComp.WallJumpX = x
			},
			ChangeState: func(state component.PlayerState) {
				stateComp.Pending = state
			},
			ChangeAnimation: func(animation string) {
				// only change to known animation defs
				def, ok := animComp.Defs[animation]
				if !ok || animComp.Sheet == nil {
					return
				}
				animComp.Current = animation
				// reset frame state to avoid using an out-of-range frame index
				animComp.Frame = 0
				animComp.FrameTimer = 0
				animComp.Playing = true
				if ok && animComp.Sheet != nil {
					rect := image.Rect(def.ColStart*def.FrameW, def.Row*def.FrameH, def.ColStart*def.FrameW+def.FrameW, def.Row*def.FrameH+def.FrameH)
					spriteComp.Image = animComp.Sheet.SubImage(rect).(*ebiten.Image)
				}
			},
			FacingLeft: func(facingLeft bool) {
				spriteComp.FacingLeft = facingLeft
			},
			CanDoubleJump: func() bool {
				if isGroundedFn != nil && isGroundedFn() {
					return false
				}
				return stateComp.JumpsUsed < 2
			},
			CanJump: func() bool {
				if isGroundedFn != nil && isGroundedFn() {
					return true
				}
				return stateComp.CoyoteTimer > 0
			},
			JumpBuffered: func() bool {
				if input.JumpPressed {
					return true
				}
				return stateComp.JumpBufferTimer > 0
			},
		}

		// update jump buffer timer: set when pressed, otherwise count down
		if input.JumpPressed {
			stateComp.JumpBufferTimer = player.JumpBufferFrames
		} else if stateComp.JumpBufferTimer > 0 {
			stateComp.JumpBufferTimer--
		}

		// update coyote timer and jump counter: reset while grounded, otherwise count down
		if ctx.IsGrounded != nil && ctx.IsGrounded() {
			stateComp.CoyoteTimer = player.CoyoteFrames
			stateComp.JumpsUsed = 0
			stateComp.WallJumpTimer = 0
		} else if stateComp.CoyoteTimer > 0 {
			stateComp.CoyoteTimer--
		}

		if stateComp.State == nil {
			stateComp.State = playerStateIdle
			stateComp.State.Enter(&ctx)
		}

		stateComp.State.HandleInput(&ctx)
		stateComp.State.Update(&ctx)

		if stateComp.Pending != nil && stateComp.Pending != stateComp.State {
			stateComp.State.Exit(&ctx)
			stateComp.State = stateComp.Pending
			stateComp.Pending = nil
			// clear buffered jump when actually performing a jump to avoid double-trigger
			if stateComp.State != nil {
				switch stateComp.State.Name() {
				case "jump":
					stateComp.JumpBufferTimer = 0
					if stateComp.JumpsUsed < 1 {
						stateComp.JumpsUsed = 1
					}
				case "double_jump":
					stateComp.JumpBufferTimer = 0
					if stateComp.JumpsUsed < 2 {
						stateComp.JumpsUsed = 2
					}
				}
			}
			stateComp.State.Enter(&ctx)
		}

		bodyComp.Body.SetAngle(0)
		bodyComp.Body.SetAngularVelocity(0)

		ecs.Add(w, e, component.AnimationComponent, animComp)
		ecs.Add(w, e, component.SpriteComponent, spriteComp)
		ecs.Add(w, e, component.PlayerStateMachineComponent, stateComp)
	}
}
