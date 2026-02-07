package components

// InputState mirrors the current input state.
type InputState struct {
	MoveX            float32
	JumpPressed      bool
	JumpHeld         bool
	AimPressed       bool
	AimHeld          bool
	MouseLeftPressed bool
	MouseWorldX      float64
	MouseWorldY      float64
	DashPressed      bool
	LastAimAngle     float64
	LastAimValid     bool
}
