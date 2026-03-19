package component

type BreakableWall struct {
	LayerName string
}

var BreakableWallComponent = NewComponent[BreakableWall]()
