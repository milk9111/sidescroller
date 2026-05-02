package component

// PlayerCheckpoint stores the last activated shrine checkpoint state.
type PlayerCheckpoint struct {
	Level       string
	X           float64
	Y           float64
	FacingLeft  bool
	Health      int
	HealUses    int
	Initialized bool
}

var PlayerCheckpointComponent = NewComponent[PlayerCheckpoint]()