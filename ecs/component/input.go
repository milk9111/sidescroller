package component

// Input stores per-frame input state for an entity.
type Input struct {
	MoveX       float64
	Jump        bool
	JumpPressed bool
	Aim         bool
}

var InputComponent = NewComponent[Input]()
