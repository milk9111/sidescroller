package component

// SpriteShake applies a temporary random render offset to sprites.
// A system updates the offset each frame while Frames remains positive.
type SpriteShake struct {
	Frames    int
	Intensity float64
	OffsetX   float64
	OffsetY   float64
}

var SpriteShakeComponent = NewComponent[SpriteShake]()
