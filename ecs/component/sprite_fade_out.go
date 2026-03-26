package component

// SpriteFadeOut linearly fades a sprite toward full transparency over time.
// Alpha is consumed by the renderer while the system advances the frame state.
type SpriteFadeOut struct {
	Frames      int
	TotalFrames int
	Alpha       float64
}

var SpriteFadeOutComponent = NewComponent[SpriteFadeOut]()
