package system

import (
	"github.com/milk9111/sidescroller/ecs/component"
)

// Player state singletons (avoid allocations on transitions).
var (
	playerStateIdle   component.PlayerState = &playerIdleState{}
	playerStateRun    component.PlayerState = &playerRunState{}
	playerStateJump   component.PlayerState = &playerJumpState{}
	playerStateDJmp   component.PlayerState = &playerDoubleJumpState{}
	playerStateWall   component.PlayerState = &playerWallGrabState{}
	playerStateFall   component.PlayerState = &playerFallState{}
	playerStateAim    component.PlayerState = &playerAimState{}
	playerStateSwing  component.PlayerState = &playerSwingState{}
	playerStateAttack component.PlayerState = &playerAttackState{}
	playerStateHit    component.PlayerState = &playerHitState{}
	playerStateDeath  component.PlayerState = &playerDeathState{}
)

type playerIdleState struct{}

type playerRunState struct{}

type playerJumpState struct{}

type playerDoubleJumpState struct{}

type playerWallGrabState struct{}

type playerFallState struct{}

type playerAimState struct{}

type playerSwingState struct{}

type playerAttackState struct{}

type playerHitState struct{}

type playerDeathState struct{}

func (playerSwingState) Name() string { return "swing" }
func (playerSwingState) Enter(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}
	// Use a dedicated swing animation if present, otherwise idle
	ctx.ChangeAnimation("swing")
}
func (playerSwingState) Exit(ctx *component.PlayerStateContext) {}
func (playerSwingState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.ChangeState == nil || ctx.Input == nil {
		return
	}

	// Swing only applies while attached via pin. If the anchor switched back to
	// slide (ground or wall contact), return to normal locomotion states.
	if ctx.IsAnchorPinned != nil && !ctx.IsAnchorPinned() {
		if ctx.IsGrounded != nil && ctx.IsGrounded() {
			if ctx.Input.MoveX == 0 {
				ctx.ChangeState(playerStateIdle)
			} else {
				ctx.ChangeState(playerStateRun)
			}
			return
		}

		ctx.ChangeState(playerStateFall)
		return
	}

	// If the player presses the aim button again while swinging, some
	// gamepad mappings also report this as an attack press. In that case
	// detach the anchor and go back to aim instead of performing an
	// attack.
	if ctx.Input.Aim && (ctx.AllowAnchor == nil || ctx.AllowAnchor()) {
		if ctx.DetachAnchor != nil {
			ctx.DetachAnchor()
		}
		ctx.ChangeState(playerStateAim)
		return
	}
	jumpReq := ctx.Input.JumpPressed
	if !jumpReq && ctx.JumpBuffered != nil {
		jumpReq = ctx.JumpBuffered()
	}
	if jumpReq {
		// Jumping out of swing should not inherit wall-jump impulse state.
		if ctx.SetWallJumpTimer != nil {
			ctx.SetWallJumpTimer(0)
		}
		if ctx.SetWallJumpX != nil {
			ctx.SetWallJumpX(0)
		}
		if ctx.DetachAnchor != nil {
			ctx.DetachAnchor()
		}
		ctx.ChangeState(playerStateJump)
		return
	}

	// If anchor was released, transition out based on grounded state
	if ctx.IsAnchored == nil || !ctx.IsAnchored() {
		if ctx.IsGrounded != nil && ctx.IsGrounded() {
			if ctx.Input.MoveX == 0 {
				ctx.ChangeState(playerStateIdle)
			} else {
				ctx.ChangeState(playerStateRun)
			}
		} else {
			ctx.ChangeState(playerStateFall)
		}
		return
	}
}
func (playerSwingState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil {
		return
	}
	anchored := false
	if ctx.IsAnchored != nil {
		anchored = ctx.IsAnchored()
	}

	// While anchored, apply a small horizontal force to pump the swing.
	if anchored && ctx.Input.MoveX != 0 && ctx.ApplyForce != nil {
		const swingPushForce = 0.04
		ctx.ApplyForce(ctx.Input.MoveX*swingPushForce, 0)
	}

	// When moving left/right while swinging we avoid directly setting
	// horizontal velocity while anchored because it fights the physics
	// constraint (slide joint) and produces reaction forces. Only apply
	// horizontal velocity when not anchored; when anchored rely on the
	// small force push above to influence swing.
	if ctx.Input.MoveX != 0 && ctx.SetVelocity != nil && ctx.GetVelocity != nil && ctx.Player != nil {
		if !anchored {
			x, y := ctx.GetVelocity()
			// per-frame horizontal acceleration (tunable, much smaller now)
			accel := ctx.Input.MoveX * 0.12
			// cap horizontal speed to roughly the player's normal move speed
			maxSpeed := ctx.Player.MoveSpeed * 1.0
			newX := x + accel
			if newX > maxSpeed {
				newX = maxSpeed
			} else if newX < -maxSpeed {
				newX = -maxSpeed
			}
			ctx.SetVelocity(newX, y)
		}
	}

	if ctx.Input.MoveX > 0 {
		ctx.FacingLeft(false)
	} else if ctx.Input.MoveX < 0 {
		ctx.FacingLeft(true)
	}
}

func (playerIdleState) Name() string { return "idle" }
func (playerIdleState) Enter(ctx *component.PlayerStateContext) {
	ctx.ChangeAnimation("idle")
}
func (playerIdleState) Exit(ctx *component.PlayerStateContext) {}
func (playerIdleState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.Aim && (ctx.AllowAnchor == nil || ctx.AllowAnchor()) {
		ctx.ChangeState(playerStateAim)
		return
	}
	jumpReq := ctx.Input.JumpPressed
	if !jumpReq && ctx.JumpBuffered != nil {
		// only trigger a buffered jump when back on the ground
		if ctx.IsGrounded != nil && ctx.IsGrounded() && ctx.JumpBuffered() {
			jumpReq = true
		}
	}
	if jumpReq && (ctx.CanJump == nil || ctx.CanJump()) {
		ctx.ChangeState(playerStateJump)
		return
	}
	if ctx.Input.MoveX != 0 {
		ctx.ChangeState(playerStateRun)
	}
}
func (playerIdleState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	_, y := ctx.GetVelocity()
	ctx.SetVelocity(0, y)
	if ctx.IsGrounded != nil && !ctx.IsGrounded() && ctx.ChangeState != nil {
		ctx.ChangeState(playerStateFall)
	}
}

func (playerRunState) Name() string { return "run" }
func (playerRunState) Enter(ctx *component.PlayerStateContext) {
	ctx.ChangeAnimation("run")
}
func (playerRunState) Exit(ctx *component.PlayerStateContext) {}
func (playerRunState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.Aim && (ctx.AllowAnchor == nil || ctx.AllowAnchor()) {
		ctx.ChangeState(playerStateAim)
		return
	}
	jumpReq := ctx.Input.JumpPressed
	if !jumpReq && ctx.JumpBuffered != nil {
		if ctx.IsGrounded != nil && ctx.IsGrounded() && ctx.JumpBuffered() {
			jumpReq = true
		}
	}
	if jumpReq && (ctx.CanJump == nil || ctx.CanJump()) {
		ctx.ChangeState(playerStateJump)
		return
	}

	if ctx.Input.MoveX == 0 {
		ctx.ChangeState(playerStateIdle)
	} else if ctx.Input.MoveX > 0 {
		ctx.FacingLeft(false)
	} else {
		ctx.FacingLeft(true)
	}

}
func (playerRunState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	_, y := ctx.GetVelocity()
	ctx.SetVelocity(ctx.Input.MoveX*ctx.Player.MoveSpeed, y)
	if ctx.IsGrounded != nil && !ctx.IsGrounded() && ctx.ChangeState != nil {
		ctx.ChangeState(playerStateFall)
		return
	}

	if ctx.Input.MoveX != 0 {
		ctx.PlayAudio("run")
	} else {
		ctx.StopAudio("run")
	}

}

func (playerJumpState) Name() string { return "jump" }
func (playerJumpState) Enter(ctx *component.PlayerStateContext) {
	ctx.ChangeAnimation("jump")
	ctx.PlayAudio("jump")

	if ctx == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	// If a wall-jump was requested in the previous state, apply a one-shot
	// horizontal impulse now that we've transitioned into the jump state.
	if ctx.GetWallJumpTimer != nil && ctx.GetWallJumpX != nil && ctx.ApplyImpulse != nil {
		if t := ctx.GetWallJumpTimer(); t > 0 {
			x := ctx.GetWallJumpX()
			if x != 0 {
				ctx.ApplyImpulse(x, 0)
			}
			// clear the wall-jump markers so they don't re-trigger
			if ctx.SetWallJumpTimer != nil {
				ctx.SetWallJumpTimer(0)
			}
			if ctx.SetWallJumpX != nil {
				ctx.SetWallJumpX(0)
			}
		}
	}
	x, _ := ctx.GetVelocity()
	ctx.SetVelocity(x, -ctx.Player.JumpSpeed)
	if ctx.SetJumpHoldTimer != nil {
		ctx.SetJumpHoldTimer(ctx.Player.JumpHoldFrames)
	}
}
func (playerJumpState) Exit(ctx *component.PlayerStateContext) {}
func (playerJumpState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.Aim && (ctx.AllowAnchor == nil || ctx.AllowAnchor()) {
		ctx.ChangeState(playerStateAim)
		return
	}
	jumpReq := ctx.Input.JumpPressed
	if !jumpReq && ctx.JumpBuffered != nil {
		jumpReq = ctx.JumpBuffered()
	}
	if jumpReq && ctx.CanDoubleJump != nil && ctx.CanDoubleJump() {
		ctx.ChangeState(playerStateDJmp)
		return
	}
}
func (playerJumpState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	_, y := ctx.GetVelocity()
	x := ctx.Input.MoveX * ctx.Player.MoveSpeed
	// apply variable jump boost while jump is held and timer remains
	if ctx.Input.Jump && ctx.GetJumpHoldTimer != nil && ctx.SetJumpHoldTimer != nil {
		if t := ctx.GetJumpHoldTimer(); t > 0 {
			y -= ctx.Player.JumpHoldBoost
			ctx.SetJumpHoldTimer(t - 1)
		}
	}
	// wall-jump impulse is applied as a one-shot impulse; do not override
	// horizontal velocity here so impulses remain effective.
	ctx.SetVelocity(x, y)
	if ctx.WallSide != nil && ctx.WallSide() != 0 {
		if shouldWallGrab(ctx) && ctx.ChangeState != nil {
			ctx.ChangeState(playerStateWall)
			return
		}
	}
	if y > 0 && ctx.ChangeState != nil {
		ctx.ChangeState(playerStateFall)
	}

	if ctx.Input.MoveX > 0 {
		ctx.FacingLeft(false)
	} else if ctx.Input.MoveX < 0 {
		ctx.FacingLeft(true)
	}
}

func (playerFallState) Name() string { return "fall" }
func (playerFallState) Enter(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}

	ctx.ChangeAnimation("fall")
}
func (playerFallState) Exit(ctx *component.PlayerStateContext) {}
func (playerFallState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.Aim && (ctx.AllowAnchor == nil || ctx.AllowAnchor()) {
		ctx.ChangeState(playerStateAim)
		return
	}
	jumpReq := ctx.Input.JumpPressed
	if !jumpReq && ctx.JumpBuffered != nil {
		if ctx.IsGrounded != nil && ctx.IsGrounded() && ctx.JumpBuffered() {
			jumpReq = true
		}
	}
	if jumpReq && (ctx.CanJump == nil || ctx.CanJump()) {
		ctx.ChangeState(playerStateJump)
		return
	}
	if jumpReq && ctx.CanDoubleJump != nil && ctx.CanDoubleJump() {
		ctx.ChangeState(playerStateDJmp)
		return
	}
	if shouldWallGrab(ctx) {
		ctx.ChangeState(playerStateWall)
		return
	}
}
func (playerFallState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	x := ctx.Input.MoveX * ctx.Player.MoveSpeed
	_, y := ctx.GetVelocity()
	if ctx.IsAnchored != nil && ctx.IsAnchored() {
		return
	}
	ctx.SetVelocity(x, y)
	if shouldWallGrab(ctx) && ctx.ChangeState != nil {
		ctx.ChangeState(playerStateWall)
		return
	}
	if ctx.IsGrounded != nil && ctx.IsGrounded() && ctx.ChangeState != nil {
		ctx.PlayAudio("land")
		if ctx.Input.MoveX == 0 {
			ctx.ChangeState(playerStateIdle)
		} else {
			ctx.ChangeState(playerStateRun)
		}
	}

	if ctx.Input.MoveX > 0 {
		ctx.FacingLeft(false)
	} else if ctx.Input.MoveX < 0 {
		ctx.FacingLeft(true)
	}
}

func (playerDoubleJumpState) Name() string { return "double_jump" }
func (playerDoubleJumpState) Enter(ctx *component.PlayerStateContext) {
	ctx.ChangeAnimation("jump")
	ctx.PlayAudio("jump")

	if ctx == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	x, _ := ctx.GetVelocity()
	ctx.SetVelocity(x, -ctx.Player.JumpSpeed)
	if ctx.SetJumpHoldTimer != nil {
		ctx.SetJumpHoldTimer(ctx.Player.JumpHoldFrames)
	}
}
func (playerDoubleJumpState) Exit(ctx *component.PlayerStateContext) {}
func (playerDoubleJumpState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.Aim && (ctx.AllowAnchor == nil || ctx.AllowAnchor()) {
		ctx.ChangeState(playerStateAim)
		return
	}
}
func (playerDoubleJumpState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	x := ctx.Input.MoveX * ctx.Player.MoveSpeed
	_, y := ctx.GetVelocity()
	// variable jump boost on double jump while held
	if ctx.Input.Jump && ctx.GetJumpHoldTimer != nil && ctx.SetJumpHoldTimer != nil {
		if t := ctx.GetJumpHoldTimer(); t > 0 {
			y -= ctx.Player.JumpHoldBoost
			ctx.SetJumpHoldTimer(t - 1)
		}
	}
	ctx.SetVelocity(x, y)
	if shouldWallGrab(ctx) && ctx.ChangeState != nil {
		ctx.ChangeState(playerStateWall)
		return
	}
	if y > 0 && ctx.ChangeState != nil {
		ctx.ChangeState(playerStateFall)
	}

	if ctx.Input.MoveX > 0 {
		ctx.FacingLeft(false)
	} else if ctx.Input.MoveX < 0 {
		ctx.FacingLeft(true)
	}
}

func (playerWallGrabState) Name() string { return "wall_grab" }
func (playerWallGrabState) Enter(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}
	ctx.ChangeAnimation("wall_grab")
	ctx.PlayAudio("land")
	if ctx.SetWallGrabTimer != nil {
		ctx.SetWallGrabTimer(ctx.Player.WallGrabFrames)
	}
}
func (playerWallGrabState) Exit(ctx *component.PlayerStateContext) {}
func (playerWallGrabState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input != nil && ctx.Input.Aim && (ctx.AllowAnchor == nil || ctx.AllowAnchor()) {
		ctx.ChangeState(playerStateAim)
		return
	}

	if ctx.Input.JumpPressed {
		if ctx.WallSide != nil && ctx.SetWallJumpTimer != nil && ctx.SetWallJumpX != nil {
			side := ctx.WallSide()
			if side == 1 {
				ctx.SetWallJumpX(ctx.Player.WallJumpPush)
			} else if side == 2 {
				ctx.SetWallJumpX(-ctx.Player.WallJumpPush)
			}
			ctx.SetWallJumpTimer(ctx.Player.WallJumpFrames)
		}
		ctx.ChangeState(playerStateJump)
		return
	}

	if ctx.IsGrounded != nil && ctx.IsGrounded() {
		if ctx.Input != nil && ctx.Input.MoveX == 0 {
			ctx.ChangeState(playerStateIdle)
		} else {
			ctx.ChangeState(playerStateRun)
		}
		return
	}
	if !shouldWallGrab(ctx) {
		ctx.ChangeState(playerStateFall)
		return
	}
}
func (playerWallGrabState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	if !shouldWallGrab(ctx) && ctx.ChangeState != nil {
		ctx.ChangeState(playerStateFall)
		return
	}
	if ctx.GetWallGrabTimer != nil && ctx.SetWallGrabTimer != nil {
		t := ctx.GetWallGrabTimer()
		if t > 0 {
			ctx.SetWallGrabTimer(t - 1)
		}
	}
	_, y := ctx.GetVelocity()
	if ctx.GetWallGrabTimer != nil && ctx.GetWallGrabTimer() > 0 {
		ctx.SetVelocity(0, 0)
	} else {
		y = ctx.Player.WallSlideSpeed
		ctx.SetVelocity(0, y)
	}
	if ctx.WallSide != nil {
		side := ctx.WallSide()
		if side == 1 {
			ctx.FacingLeft(true)
		} else if side == 2 {
			ctx.FacingLeft(false)
		}
	}
}

func (playerAimState) Name() string { return "aim" }
func (playerAimState) Enter(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}
	ctx.ChangeAnimation("idle")
}
func (playerAimState) Exit(ctx *component.PlayerStateContext) {}
func (playerAimState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if !ctx.Input.Aim || (ctx.AllowAnchor != nil && !ctx.AllowAnchor()) {
		if ctx.IsGrounded != nil && ctx.IsGrounded() {
			if ctx.Input.MoveX == 0 {
				ctx.ChangeState(playerStateIdle)
			} else {
				ctx.ChangeState(playerStateRun)
			}
			return
		}
		ctx.ChangeState(playerStateFall)
		return
	}
	jumpReq := ctx.Input.JumpPressed
	if !jumpReq && ctx.JumpBuffered != nil {
		if ctx.IsGrounded != nil && ctx.IsGrounded() && ctx.JumpBuffered() {
			jumpReq = true
		}
	}
	if jumpReq && (ctx.CanJump == nil || ctx.CanJump()) {
		ctx.ChangeState(playerStateJump)
		return
	}
	if jumpReq && ctx.CanDoubleJump != nil && ctx.CanDoubleJump() {
		ctx.ChangeState(playerStateDJmp)
		return
	}
}
func (playerAimState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}

	// When aiming mid-air while falling, slow vertical velocity instead
	// of stopping it entirely to create a "slow motion" feel.
	_, y := ctx.GetVelocity()
	if y != 0 {
		y = y * ctx.Player.AimSlowFactor
	}
	ctx.SetVelocity(0, y)
}

func shouldWallGrab(ctx *component.PlayerStateContext) bool {
	if ctx == nil {
		return false
	}
	if ctx.AllowWallGrab != nil && !ctx.AllowWallGrab() {
		return false
	}
	if ctx.WallSide == nil || ctx.Input == nil {
		return false
	}
	if ctx.IsGrounded != nil && ctx.IsGrounded() {
		return false
	}
	side := ctx.WallSide()
	if side == 1 && ctx.Input.MoveX < 0 {
		return true
	}
	if side == 2 && ctx.Input.MoveX > 0 {
		return true
	}
	return false
}

func (playerAttackState) Name() string { return "attack" }
func (playerAttackState) Enter(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}

	if ctx.IsGrounded() {
		ctx.SetVelocity(0, 0)
	}

	ctx.ChangeAnimation("attack")
	ctx.PlayAudio("attack")
}
func (playerAttackState) Exit(ctx *component.PlayerStateContext) {
	ctx.StopAudio("attack")
}
func (playerAttackState) HandleInput(ctx *component.PlayerStateContext) { return }
func (playerAttackState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.GetAnimationPlaying == nil || ctx.ChangeState == nil || ctx.Input == nil {
		return
	}

	if !ctx.GetAnimationPlaying() {
		if ctx.IsGrounded != nil && ctx.IsGrounded() {
			if ctx.Input.MoveX == 0 {
				ctx.ChangeState(playerStateIdle)
			} else {
				ctx.ChangeState(playerStateRun)
			}
		} else {
			ctx.ChangeState(playerStateFall)
		}
	}
}

func (playerHitState) Name() string { return "hit" }
func (playerHitState) Enter(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}
	ctx.ChangeAnimation("hit")
	ctx.PlayAudio("hit")

	// Add timed invulnerability and white flash while in hit state.
	if ctx.AddInvulnerable != nil {
		// default to 30 frames unless Player config provides a different value
		frames := 30
		ctx.AddInvulnerable(frames)
	}

	if ctx.AddWhiteFlash != nil {
		ctx.AddWhiteFlash(30, 5)
	}
}
func (playerHitState) Exit(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}
	// Do not remove Invulnerable here; timed invulnerability should persist
	// until the InvulnerabilitySystem expires it or other game logic removes it.
	if ctx.RemoveWhiteFlash != nil {
		ctx.RemoveWhiteFlash()
	}
}
func (playerHitState) HandleInput(ctx *component.PlayerStateContext) { return }
func (playerHitState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.GetAnimationPlaying == nil || ctx.ChangeState == nil {
		return
	}
	// When the hit animation completes, go to idle
	if !ctx.GetAnimationPlaying() {
		ctx.ChangeState(playerStateIdle)
	}
}

func (playerDeathState) Name() string { return "death" }
func (playerDeathState) Enter(ctx *component.PlayerStateContext) {
	if ctx == nil {
		return
	}
	// play death animation and stop motion
	ctx.ChangeAnimation("death")

	ctx.PlayAudio("death")

	if ctx.SetVelocity != nil && ctx.GetVelocity != nil {
		_, y := ctx.GetVelocity()
		ctx.SetVelocity(0, y)
	}
	// initialize death timer: use -1 to indicate "waiting for animation end".
	// Once the death animation finishes we'll start a short post-death delay
	// (e.g. 120 frames ~= 2s at 60fps) before requesting reload.
	if ctx.SetDeathTimer != nil {
		ctx.SetDeathTimer(-1)
	}
	// Mark invulnerable while in death state so no further damage is processed
	if ctx.PlayAudio != nil {
		// use presence of callback to access ECS via ChangeState handling in controller
	}
}
func (playerDeathState) Exit(ctx *component.PlayerStateContext)        {}
func (playerDeathState) HandleInput(ctx *component.PlayerStateContext) { return }
func (playerDeathState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.GetAnimationPlaying == nil {
		return
	}
	// keep player from moving while dead
	if ctx.SetVelocity != nil && ctx.GetVelocity != nil {
		_, y := ctx.GetVelocity()
		ctx.SetVelocity(0, y)
	}
	if ctx.GetDeathTimer != nil && ctx.SetDeathTimer != nil {
		t := ctx.GetDeathTimer()
		// sentinel -1 => wait for death animation to finish
		if t == -1 {
			if !ctx.GetAnimationPlaying() {
				// start post-death delay: ~2 seconds at 60fps
				ctx.SetDeathTimer(120)
			}
		} else if t > 0 {
			t--
			ctx.SetDeathTimer(t)
			if t == 0 {
				if ctx.RequestReload != nil {
					ctx.RequestReload()
				}
			}
		}
	}
}
