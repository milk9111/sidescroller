package system

import "github.com/milk9111/sidescroller/ecs/component"

// Player state singletons (avoid allocations on transitions).
var (
	playerStateIdle component.PlayerState = &playerIdleState{}
	playerStateRun  component.PlayerState = &playerRunState{}
	playerStateJump component.PlayerState = &playerJumpState{}
	playerStateFall component.PlayerState = &playerFallState{}
)

type playerIdleState struct{}

type playerRunState struct{}

type playerJumpState struct{}

type playerFallState struct{}

func (playerIdleState) Name() string { return "idle" }
func (playerIdleState) Enter(ctx *component.PlayerStateContext) {
	ctx.ChangeAnimation("idle")
}
func (playerIdleState) Exit(ctx *component.PlayerStateContext) {}
func (playerIdleState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.JumpPressed && (ctx.CanJump == nil || ctx.CanJump()) {
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
	if ctx.Input.JumpPressed && (ctx.CanJump == nil || ctx.CanJump()) {
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
	ctx.ChangeAnimation("idle")

	if ctx == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	x, _ := ctx.GetVelocity()
	ctx.SetVelocity(x, -ctx.Player.JumpSpeed)
}
func (playerJumpState) Exit(ctx *component.PlayerStateContext) {}
func (playerJumpState) HandleInput(ctx *component.PlayerStateContext) {
	// no-op for now
}
func (playerJumpState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	x := ctx.Input.MoveX * ctx.Player.MoveSpeed
	_, y := ctx.GetVelocity()
	ctx.SetVelocity(x, y)
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
	ctx.ChangeAnimation("idle")
}
func (playerFallState) Exit(ctx *component.PlayerStateContext) {}
func (playerFallState) HandleInput(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.ChangeState == nil {
		return
	}
	if ctx.Input.JumpPressed && (ctx.CanJump == nil || ctx.CanJump()) {
		ctx.ChangeState(playerStateJump)
		return
	}
}
func (playerFallState) Update(ctx *component.PlayerStateContext) {
	if ctx == nil || ctx.Input == nil || ctx.SetVelocity == nil || ctx.GetVelocity == nil {
		return
	}
	x := ctx.Input.MoveX * ctx.Player.MoveSpeed
	_, y := ctx.GetVelocity()
	ctx.SetVelocity(x, y)
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
