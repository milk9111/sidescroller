package component

type Player struct {
	MoveSpeed float64
	JumpSpeed float64
	// JumpHoldFrames: maximum frames jump boost is applied while holding jump
	JumpHoldFrames int
	// JumpHoldBoost: additional upward velocity applied per frame while holding
	JumpHoldBoost    float64
	CoyoteFrames     int
	WallGrabFrames   int
	WallSlideSpeed   float64
	WallJumpPush     float64
	WallJumpFrames   int
	JumpBufferFrames int
}

var PlayerComponent = NewComponent[Player]()
