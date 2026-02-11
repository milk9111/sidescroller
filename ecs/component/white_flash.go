package component

// WhiteFlash makes a sprite render as full white while active. Timing is
// frame-based and managed by systems (e.g. player controller).
type WhiteFlash struct {
	// Frames remaining for the whole flash effect (in update ticks)
	Frames int
	// Interval in frames between toggles of the white-on state
	Interval int
	// internal timer (frames) used to count toward the next toggle
	Timer int
	// On determines whether the sprite should currently be rendered white
	On bool
}

var WhiteFlashComponent = NewComponent[WhiteFlash]()
