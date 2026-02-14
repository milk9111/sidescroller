package component

// TransitionPop is a one-shot component placed on the player after a
// transition spawn. A system applies an initial launch velocity, then keeps
// this component present to lock player control/state updates until grounded.
type TransitionPop struct {
	VX          float64
	VY          float64
	FacingLeft  bool
	WallJumpDur int
	WallJumpX   float64
	Applied     bool
}

var TransitionPopComponent = NewComponent[TransitionPop]()
