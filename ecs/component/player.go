package component

type Player struct {
	MoveSpeed float64
	JumpSpeed float64
}

var PlayerComponent = NewComponent[Player]()
