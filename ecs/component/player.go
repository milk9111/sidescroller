package component

type Player struct {
	MoveSpeed            float64
	JumpSpeed            float64
	JumpHoldFrames       int
	JumpHoldBoost        float64
	FallMultiplier       float64
	CoyoteFrames         int
	WallGrabFrames       int
	WallSlideSpeed       float64
	WallJumpPush         float64
	WallJumpFrames       int
	JumpBufferFrames     int
	AnchorReelSpeed      float64
	AnchorMinLength      float64
	AimSlowFactor        float64
	HitFreezeFrames      int
	DamageShakeIntensity float64
}

var PlayerComponent = NewComponent[Player]()
