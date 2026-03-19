package component

type AreaTileStampRotationMode string

const (
	AreaTileStampRotationNone            AreaTileStampRotationMode = "none"
	AreaTileStampRotationTransitionEnter AreaTileStampRotationMode = "transition_enter_dir"
	AreaTileStampRotationOpenNeighbor    AreaTileStampRotationMode = "open_neighbor"
)

// AreaTileStamp marks an entity sprite as a per-tile visual that should be
// repeated across AreaBounds without changing the authored bounds or physics.
type AreaTileStamp struct {
	TileWidth      float64
	TileHeight     float64
	RotationMode   AreaTileStampRotationMode
	RotationOffset float64
}

var AreaTileStampComponent = NewComponent[AreaTileStamp]()
