package system

import "github.com/milk9111/sidescroller/ecs/component"

// Player state singletons (avoid allocations on transitions).
var (
	playerStateIdle component.PlayerState = &playerIdleState{}
	playerStateRun  component.PlayerState = &playerRunState{}
	playerStateJump component.PlayerState = &playerJumpState{}
	playerStateDJmp component.PlayerState = &playerDoubleJumpState{}
	playerStateWall component.PlayerState = &playerWallGrabState{}
	playerStateFall component.PlayerState = &playerFallState{}
	playerStateAim  component.PlayerState = &playerAimState{}
)

type playerIdleState struct{}

type playerRunState struct{}

type playerJumpState struct{}

type playerDoubleJumpState struct{}

type playerWallGrabState struct{}

type playerFallState struct{}

type playerAimState struct{}

func (playerIdleState) Name() string { return "idle" }
func (playerIdleState) Enter(ctx *component.PlayerStateContext) {
	ctx.ChangeAnimation("idle")
}
func (playerIdleState) Exit(ctx *component.PlayerStateContext) {}
func (playerIdleState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.Aim {
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
	if ctx.Input.Aim {
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
	}
}

func (playerJumpState) Name() string { return "jump" }
func (playerJumpState) Enter(ctx *component.PlayerStateContext) {
	ctx.ChangeAnimation("jump")

	if ctx == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
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
	if ctx.Input.Aim {
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
	if ctx.GetWallJumpTimer != nil && ctx.SetWallJumpTimer != nil && ctx.GetWallJumpX != nil {
		if t := ctx.GetWallJumpTimer(); t > 0 {
			x = ctx.GetWallJumpX()
			ctx.SetWallJumpTimer(t - 1)
		}
	}
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
	if ctx.Input.Aim {
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
	if ctx.Input.Aim {
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
	if ctx.SetWallGrabTimer != nil {
		ctx.SetWallGrabTimer(ctx.Player.WallGrabFrames)
	}
}
func (playerWallGrabState) Exit(ctx *component.PlayerStateContext) {}
func (playerWallGrabState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input != nil && ctx.Input.Aim {
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
	if !ctx.Input.Aim {
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
	if ctx == nil || ctx.WallSide == nil || ctx.Input == nil {
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
