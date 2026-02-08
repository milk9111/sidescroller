package component

// LevelBounds stores the world-space bounds of the current level.
type LevelBounds struct {
	Width  float64
	Height float64
}

var LevelBoundsComponent = NewComponent[LevelBounds]()
