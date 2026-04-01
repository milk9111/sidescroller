package component

// Abilities defines which optional player abilities are enabled.
type Abilities struct {
	DoubleJump bool
	WallGrab   bool
	Anchor     bool
	Heal       bool
}

var AbilitiesComponent = NewComponent[Abilities]()
