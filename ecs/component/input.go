package component

// Input stores per-frame input state for an entity.
type Input struct {
	Disabled            bool
	MoveX               float64
	Jump                bool
	JumpPressed         bool
	Aim                 bool
	AimX                float64
	AimY                float64
	LookY               float64
	AnchorPressed       bool
	AttackPressed       bool
	UpwardAttackPressed bool
}

var InputComponent = NewComponent[Input]()
