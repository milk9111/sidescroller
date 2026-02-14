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
	Input               *Input
	Player              *Player
	GetVelocity         func() (x, y float64)
	SetVelocity         func(x, y float64)
	ApplyForce          func(x, y float64)
	SetAngle            func(angle float64)
	SetAngularVelocity  func(omega float64)
	IsGrounded          func() bool
	IsAnchored          func() bool
	IsAnchorPinned      func() bool
	WallSide            func() int
	GetWallGrabTimer    func() int
	SetWallGrabTimer    func(frames int)
	GetWallJumpTimer    func() int
	SetWallJumpTimer    func(frames int)
	GetWallJumpX        func() float64
	SetWallJumpX        func(x float64)
	GetJumpHoldTimer    func() int
	SetJumpHoldTimer    func(frames int)
	ChangeState         func(state PlayerState)
	ChangeAnimation     func(animation string)
	DetachAnchor        func()
	FacingLeft          func(facingLeft bool)
	CanDoubleJump       func() bool
	JumpBuffered        func() bool
	CanJump             func() bool
	GetAnimationPlaying func() bool
	GetDeathTimer       func() int
	SetDeathTimer       func(frames int)
	RequestReload       func()
	PlayAudio           func(name string)
	StopAudio           func(name string)
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
	// JumpsUsed counts jumps since last grounded (0 = none, 1 = jumped, 2 = double jumped)
	JumpsUsed int
	// WallGrabTimer counts frames remaining to stick to wall before sliding.
	WallGrabTimer int
	// WallJumpTimer counts frames to apply wall jump horizontal impulse.
	WallJumpTimer int
	// WallJumpX is the horizontal velocity used during wall jump impulse.
	WallJumpX float64
	// JumpHoldTimer counts frames remaining to apply extra upward boost while
	// the jump button is held (variable jump height).
	JumpHoldTimer int
	// DeathTimer counts frames remaining until a reload should be requested
	// once the player has entered the death state.
	DeathTimer int
}

var PlayerStateMachineComponent = NewComponent[PlayerStateMachine]()
