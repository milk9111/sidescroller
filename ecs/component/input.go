package component

// Input stores per-frame input state for an entity.
type Input struct {
	MoveX       float64
	Jump        bool
	JumpPressed bool
	Aim         bool
	AimX        float64
	AimY        float64
	// LookY is a small vertical look input used to offset the camera.
	// Negative = look up, Positive = look down.
	LookY         float64
	AnchorPressed bool
	AttackPressed bool
}

var InputComponent = NewComponent[Input]()
