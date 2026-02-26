package component

// LevelLoaded is a marker added when a level load + placement has completed.
// Sequence increments each load so systems can detect transitions/reloads
// even when the world pointer is reused.
type LevelLoaded struct {
	Sequence uint64
}

var LevelLoadedComponent = NewComponent[LevelLoaded]()
