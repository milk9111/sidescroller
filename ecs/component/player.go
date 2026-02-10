package component

type Player struct {
	MoveSpeed        float64
	JumpSpeed        float64
	JumpHoldFrames   int
	JumpHoldBoost    float64
	CoyoteFrames     int
	WallGrabFrames   int
	WallSlideSpeed   float64
	WallJumpPush     float64
	WallJumpFrames   int
	JumpBufferFrames int
	AimSlowFactor    float64
}

var PlayerComponent = NewComponent[Player]()
