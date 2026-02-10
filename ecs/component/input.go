package component

// Input stores per-frame input state for an entity.
type Input struct {
	MoveX         float64
	Jump          bool
	JumpPressed   bool
	Aim           bool
	AimX          float64
	AimY          float64
	AnchorPressed bool
}

var InputComponent = NewComponent[Input]()
