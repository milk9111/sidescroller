package component

type AreaTileStampRotationMode string

type AreaTileStampOverdrawMode string

type AreaTileStampSide string

const (
	AreaTileStampRotationNone            AreaTileStampRotationMode = "none"
	AreaTileStampRotationTransitionEnter AreaTileStampRotationMode = "transition_enter_dir"
	AreaTileStampRotationOpenNeighbor    AreaTileStampRotationMode = "open_neighbor"

	AreaTileStampOverdrawNone            AreaTileStampOverdrawMode = "none"
	AreaTileStampOverdrawAll             AreaTileStampOverdrawMode = "all"
	AreaTileStampOverdrawNonPlayerFacing AreaTileStampOverdrawMode = "non_player_facing"

	AreaTileStampSideNone   AreaTileStampSide = "none"
	AreaTileStampSideTop    AreaTileStampSide = "top"
	AreaTileStampSideRight  AreaTileStampSide = "right"
	AreaTileStampSideBottom AreaTileStampSide = "bottom"
	AreaTileStampSideLeft   AreaTileStampSide = "left"
)

// AreaTileStamp marks an entity sprite as a per-tile visual that should be
// repeated across AreaBounds without changing the authored bounds or physics.
type AreaTileStamp struct {
	TileWidth        float64
	TileHeight       float64
	Overdraw         float64
	OverdrawMode     AreaTileStampOverdrawMode
	PlayerFacingSide AreaTileStampSide
	RotationMode     AreaTileStampRotationMode
	RotationOffset   float64
}

var AreaTileStampComponent = NewComponent[AreaTileStamp]()
