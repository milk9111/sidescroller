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
			vel := bodyComp.Body.Velocity()
			return math.Abs(vel.Y) < groundedEpsilon
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
			CanJump: func() bool {
				if isGroundedFn != nil && isGroundedFn() {
					return true
				}
				return stateComp.CoyoteTimer > 0
			},
		}

		// update coyote timer: reset while grounded, otherwise count down
		if ctx.IsGrounded != nil && ctx.IsGrounded() {
			stateComp.CoyoteTimer = player.CoyoteFrames
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
			stateComp.State.Enter(&ctx)
		}

		bodyComp.Body.SetAngle(0)
		bodyComp.Body.SetAngularVelocity(0)

		ecs.Add(w, e, component.AnimationComponent, animComp)
		ecs.Add(w, e, component.SpriteComponent, spriteComp)
		ecs.Add(w, e, component.PlayerStateMachineComponent, stateComp)
	}
}
