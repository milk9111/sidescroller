package component

// LevelLayerStateMap stores per-level authored layer activation state so
// toggles survive transitions and can be serialized into save files.
type LevelLayerStateMap struct {
	States map[string]bool
}

var LevelLayerStateMapComponent = NewComponent[LevelLayerStateMap]()
