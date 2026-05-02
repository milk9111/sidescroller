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
	Input                   *Input
	Player                  *Player
	GetPosition             func() (x, y float64)
	GetVelocity             func() (x, y float64)
	SetPosition             func(x, y float64)
	SetVelocity             func(x, y float64)
	SetGravityScale         func(scale float64)
	ApplyForce              func(x, y float64)
	SetAngle                func(angle float64)
	SetAngularVelocity      func(omega float64)
	IsGrounded              func() bool
	IsAnchored              func() bool
	IsAnchorPinned          func() bool
	CanClamber              func() bool
	GetClamberTarget        func() (x, y float64)
	GetClamberFrames        func() int
	SetClamberFrames        func(frames int)
	GetClamberStart         func() (x, y float64)
	SetClamberStart         func(x, y float64)
	GetStoredClamberTarget  func() (x, y float64)
	SetStoredClamberTarget  func(x, y float64)
	WallSide                func() int
	GetWallGrabTimer        func() int
	SetWallGrabTimer        func(frames int)
	GetWallJumpTimer        func() int
	SetWallJumpTimer        func(frames int)
	GetWallJumpX            func() float64
	SetWallJumpX            func(x float64)
	ApplyImpulse            func(x, y float64)
	GetJumpHoldTimer        func() int
	SetJumpHoldTimer        func(frames int)
	GetFallFrames           func() int
	SetFallFrames           func(frames int)
	ChangeState             func(state PlayerState)
	ChangeAnimation         func(animation string)
	DetachAnchor            func()
	FacingLeft              func(facingLeft bool)
	CanDoubleJump           func() bool
	AllowDoubleJump         func() bool
	AllowWallGrab           func() bool
	AllowAnchor             func() bool
	JumpBuffered            func() bool
	CanJump                 func() bool
	DisablePlayerCollisions func()
	RestorePlayerCollisions func()
	GetAnimationDuration    func(animation string) int
	GetAnimationPlaying     func() bool
	GetDeathTimer           func() int
	SetDeathTimer           func(frames int)
	CompleteShrineHeal      func()
	BeginCheckpointRespawn  func()
	RequestReload           func()
	PlayAudio               func(name string)
	StopAudio               func(name string)
	ConsumeHitEvent         func() bool
	AddInvulnerable         func(frames int)
	RemoveInvulnerable      func()
	AddWhiteFlash           func(frames int, interval int)
	RemoveWhiteFlash        func()
	TryHeal                 func(amount int, maxUses int) bool
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
	// HealUses counts number of heal actions consumed.
	HealUses int
	// FallFrames counts consecutive updates spent in the fall state.
	FallFrames int
	// ClamberFramesElapsed counts frames spent in the active clamber move.
	ClamberFramesElapsed int
	// ClamberStartX/Y is the body center at the moment clamber starts.
	ClamberStartX float64
	ClamberStartY float64
	// ClamberTargetX/Y is the body center to place the player at when clamber completes.
	ClamberTargetX float64
	ClamberTargetY float64
	// ClamberCollisionCategory/Mask stores the player's collision filter while clamber temporarily disables collisions.
	ClamberCollisionCategory uint32
	ClamberCollisionMask     uint32
	ClamberCollisionSaved    bool
}

var PlayerStateMachineComponent = NewComponent[PlayerStateMachine]()
