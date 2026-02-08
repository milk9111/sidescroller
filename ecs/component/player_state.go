package component

// PlayerState defines the interface for player state machine states.
// Each state owns its own enter/exit, input handling, and update logic.
type PlayerState interface {
	Name() string
	Enter(ctx *PlayerStateContext)
	Exit(ctx *PlayerStateContext)
	HandleInput(ctx *PlayerStateContext)
	Update(ctx *PlayerStateContext)
}

// PlayerStateContext provides controlled access to input and physics for a state.
// It intentionally uses callbacks to avoid tight coupling to the ECS package.
type PlayerStateContext struct {
	Input              *Input
	Player             *Player
	GetVelocity        func() (x, y float64)
	SetVelocity        func(x, y float64)
	SetAngle           func(angle float64)
	SetAngularVelocity func(omega float64)
	IsGrounded         func() bool
	ChangeState        func(state PlayerState)
	ChangeAnimation    func(animation string)
	FacingLeft         func(facingLeft bool)
	// JumpBuffered reports whether a recent jump press should still trigger a jump
	// (pressed within jump-buffer frames).
	JumpBuffered func() bool
	// CanJump reports whether a jump should be allowed (grounded or within coyote time)
	CanJump func() bool
}

// PlayerStateMachine stores the active and pending states for the player.
type PlayerStateMachine struct {
	State   PlayerState
	Pending PlayerState
	// CoyoteTimer counts frames remaining where a jump is allowed after leaving ground
	CoyoteTimer int
	// JumpBufferTimer counts frames remaining after a jump press where a jump
	// should be triggered once grounded.
	JumpBufferTimer int
}

var PlayerStateMachineComponent = NewComponent[PlayerStateMachine]()
