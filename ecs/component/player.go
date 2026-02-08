package component

type Player struct {
	MoveSpeed    float64
	JumpSpeed    float64
	CoyoteFrames int
}

var PlayerComponent = NewComponent[Player]()
