package component

// SafeRespawn stores the last grounded-safe position for an entity.
type SafeRespawn struct {
	X           float64
	Y           float64
	Initialized bool
}

var SafeRespawnComponent = NewComponent[SafeRespawn]()
