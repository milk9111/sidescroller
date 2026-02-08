package component

type Player struct {
	MoveSpeed    float64
	JumpSpeed    float64
	CoyoteFrames int
	// JumpBufferFrames allows a short window after pressing jump before landing
	// where a jump will still be triggered when grounded.
	JumpBufferFrames int
}

var PlayerComponent = NewComponent[Player]()
