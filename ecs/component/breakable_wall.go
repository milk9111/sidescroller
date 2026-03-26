package component

type BreakableWall struct {
	LayerName             string
	DestroyedSignalTarget string
}

var BreakableWallComponent = NewComponent[BreakableWall]()
