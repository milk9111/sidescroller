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

	ecs.ForEach7(w,
		component.PlayerComponent.Kind(),
		component.InputComponent.Kind(),
		component.PhysicsBodyComponent.Kind(),
		component.PlayerStateMachineComponent.Kind(),
		component.AnimationComponent.Kind(),
		component.SpriteComponent.Kind(),
		component.AudioComponent.Kind(),
		func(e ecs.Entity, player *component.Player, input *component.Input, bodyComp *component.PhysicsBody, stateComp *component.PlayerStateMachine, animComp *component.Animation, spriteComp *component.Sprite, audioComp *component.Audio) {
			if bodyComp.Body == nil {
				return
			}

			// While transition pop is active, lock player input/state updates.
			if pop, ok := ecs.Get(w, e, component.TransitionPopComponent.Kind()); ok && pop != nil {
				bodyComp.Body.SetAngle(0)
				bodyComp.Body.SetAngularVelocity(0)
				return
			}

			// Consume any one-shot state interrupt events (e.g. from combat)
			if irq, ok := ecs.Get(w, e, component.PlayerStateInterruptComponent.Kind()); ok {
				switch irq.State {
				case "hit":
					stateComp.Pending = playerStateHit
				case "death":
					stateComp.Pending = playerStateDeath
				}
				_ = ecs.Remove(w, e, component.PlayerStateInterruptComponent.Kind())
			}

			// helper ground check used by multiple closures
			isGroundedFn := func() bool {
				// Prefer chipmunk-derived grounded state when available.
				if pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent.Kind()); ok {
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

				isGrounded := false
				ecs.ForEach(w, component.PhysicsBodyComponent.Kind(), func(other ecs.Entity, oComp *component.PhysicsBody) {
					if other == e || oComp.Body == nil || isGrounded {
						return
					}

					oPos := oComp.Body.Position()
					oHalfW := oComp.Width / 2.0
					oHalfH := oComp.Height / 2.0
					otherTop := oPos.Y - oHalfH

					// horizontal overlap check
					if math.Abs(pos.X-oPos.X) > (halfW + oHalfW) {
						return
					}

					// is player's bottom within groundCheckDist of other's top?
					if playerBottom >= otherTop-groundCheckDist && playerBottom <= otherTop+groundCheckDist {
						isGrounded = true
						return
					}
				})

				return isGrounded
			}

			ctx := component.PlayerStateContext{
				Input:  input,
				Player: player,
				GetVelocity: func() (x, y float64) {
					vel := bodyComp.Body.Velocity()
					return vel.X, vel.Y
				},
				SetVelocity: func(x, y float64) {
					bodyComp.Body.SetVelocityVector(cp.Vector{X: x, Y: y})
				},
				ApplyForce: func(x, y float64) {
					bodyComp.Body.ApplyForceAtWorldPoint(cp.Vector{X: x, Y: y}, bodyComp.Body.Position())
				},
				SetAngle: func(angle float64) {
					bodyComp.Body.SetAngle(angle)
				},
				SetAngularVelocity: func(omega float64) {
					bodyComp.Body.SetAngularVelocity(omega)
				},
				IsGrounded: isGroundedFn,
				IsAnchored: func() bool {
					isAnchored := false
					ecs.ForEach2(w, component.AnchorJointComponent.Kind(), component.AnchorTagComponent.Kind(), func(e ecs.Entity, aj *component.AnchorJoint, _ *component.AnchorTag) {
						if aj.Slide != nil || aj.Pivot != nil || aj.Pin != nil {
							isAnchored = true
						}
					})

					return isAnchored
				},
				IsAnchorPinned: func() bool {
					isPinned := false
					ecs.ForEach2(w, component.AnchorJointComponent.Kind(), component.AnchorTagComponent.Kind(), func(e ecs.Entity, aj *component.AnchorJoint, _ *component.AnchorTag) {
						if aj.Pin != nil {
							isPinned = true
						}
					})
					return isPinned
				},
				WallSide: func() int {
					if pc, ok := ecs.Get(w, e, component.PlayerCollisionComponent.Kind()); ok {
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
				GetAnimationPlaying: func() bool {
					return animComp.Playing
				},
				GetDeathTimer: func() int {
					return stateComp.DeathTimer
				},
				SetDeathTimer: func(frames int) {
					stateComp.DeathTimer = frames
				},
				RequestReload: func() {
					// create a one-shot reload request entity
					if _, ok := ecs.First(w, component.ReloadRequestComponent.Kind()); !ok {
						req := ecs.CreateEntity(w)
						_ = ecs.Add(w, req, component.ReloadRequestComponent.Kind(), &component.ReloadRequest{})
					}
				},
					ConsumeHitEvent: func() bool {
						if ecs.Has(w, e, component.HitEventComponent.Kind()) {
							_ = ecs.Remove(w, e, component.HitEventComponent.Kind())
							return true
						}
						return false
					},
				DetachAnchor: func() {
					// find any anchor with an active joint and mark it pending-destroy
					ecs.ForEach2(w, component.AnchorJointComponent.Kind(), component.AnchorTagComponent.Kind(), func(e ecs.Entity, aj *component.AnchorJoint, _ *component.AnchorTag) {
						if aj.Slide != nil || aj.Pivot != nil || aj.Pin != nil {
							_ = ecs.Add(w, e, component.AnchorPendingDestroyComponent.Kind(), &component.AnchorPendingDestroy{})
						}
					})
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
				PlayAudio: func(name string) {
					for i, s := range audioComp.Names {
						if s == name {
							audioComp.Play[i] = true
							return
						}
					}
				},
				StopAudio: func(name string) {
					for i, s := range audioComp.Names {
						if s == name {
							audioComp.Stop[i] = true
							return
						}
					}
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

			// If player is anchored via pin joint, prefer swing unless we're already in
			// a state that should take precedence (jump, double jump, wall grab, aim).
			if ctx.IsAnchorPinned != nil && ctx.IsAnchorPinned() {
				curr := ""
				if stateComp.State != nil {
					curr = stateComp.State.Name()
				}
				if curr != "swing" && curr != "jump" && curr != "double_jump" && curr != "wall_grab" && curr != "aim" {
					stateComp.Pending = playerStateSwing
				}
			}

			// Allow immediate attack transitions from input
			if input.AttackPressed {
				if stateComp.State == nil || stateComp.State.Name() != "attack" {
					stateComp.Pending = playerStateAttack
				}
			}
			stateComp.State.HandleInput(&ctx)
			stateComp.State.Update(&ctx)

			if stateComp.Pending != nil && stateComp.Pending != stateComp.State {
				prev := stateComp.State
				if prev != nil {
					prev.Exit(&ctx)
				}
				stateComp.State = stateComp.Pending
				stateComp.Pending = nil
				// If we transitioned away from the hit state, remove invulnerability
				if prev != nil && prev.Name() == "hit" {
					_ = ecs.Remove(w, e, component.InvulnerableComponent.Kind())
				}
				// If entering hit state, add invulnerability while the animation plays
				if stateComp.State != nil && stateComp.State.Name() == "hit" {
					_ = ecs.Add(w, e, component.InvulnerableComponent.Kind(), &component.Invulnerable{})
					// also add a white flash effect: flash for ~30 frames, toggling every 5 frames
					_ = ecs.Add(w, e, component.WhiteFlashComponent.Kind(), &component.WhiteFlash{Frames: 30, Interval: 5, Timer: 0, On: true})
				}
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
					case "wall_grab", "swing":
						stateComp.JumpsUsed = 0
					}
				}
				if stateComp.State != nil {
					stateComp.State.Enter(&ctx)
				}
			}

			// If player is anchored, allow natural rotation for swinging.
			skipClamp := false
			if aj, ok := ecs.Get(w, e, component.AnchorJointComponent.Kind()); ok {
				if aj.Pivot != nil || aj.Pin != nil {
					skipClamp = true
				}
			}
			if !skipClamp {
				bodyComp.Body.SetAngle(0)
				bodyComp.Body.SetAngularVelocity(0)
			}

			ecs.Add(w, e, component.AnimationComponent.Kind(), animComp)
			ecs.Add(w, e, component.SpriteComponent.Kind(), spriteComp)
			ecs.Add(w, e, component.PlayerStateMachineComponent.Kind(), stateComp)
		},
	)
}
