package component

// Input stores per-frame input state for an entity.
type Input struct {
	AimX                 float64
	AimY                 float64
	LookY                float64
	MoveX                float64
	Disabled             bool
	Jump                 bool
	JumpPressed          bool
	Aim                  bool
	AnchorPressed        bool
	AutoAnchorPressed    bool
	AnchorReelIn         bool
	AnchorReelOut        bool
	AttackPressed        bool
	UpwardAttackPressed  bool
	HealPressed          bool
	AnchorReleasePressed bool
	UsingGamepad         bool

	MouseDoubleClickPressedTimer int
}

var InputComponent = NewComponent[Input]()
