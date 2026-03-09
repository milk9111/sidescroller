package component

// LevelGrid stores runtime cell metadata for the currently loaded level.
type LevelGrid struct {
	Width    int
	Height   int
	TileSize float64
	Occupied []bool
	Solid    []bool
}

func (g *LevelGrid) InBounds(cellX, cellY int) bool {
	if g == nil {
		return false
	}
	return cellX >= 0 && cellY >= 0 && cellX < g.Width && cellY < g.Height
}

func (g *LevelGrid) CellIndex(cellX, cellY int) int {
	if !g.InBounds(cellX, cellY) {
		return -1
	}
	return cellY*g.Width + cellX
}

func (g *LevelGrid) CellOccupied(cellX, cellY int) bool {
	idx := g.CellIndex(cellX, cellY)
	return idx >= 0 && idx < len(g.Occupied) && g.Occupied[idx]
}

func (g *LevelGrid) CellSolid(cellX, cellY int) bool {
	idx := g.CellIndex(cellX, cellY)
	return idx >= 0 && idx < len(g.Solid) && g.Solid[idx]
}

var LevelGridComponent = NewComponent[LevelGrid]()
