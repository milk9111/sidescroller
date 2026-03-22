package component

import "github.com/milk9111/sidescroller/levels"

// LevelRuntime stores authored level data needed for runtime layer toggles.
type LevelRuntime struct {
	Name         string
	Level        *levels.Level
	TileSize     float64
	LoadedLayers []bool
}

var LevelRuntimeComponent = NewComponent[LevelRuntime]()
