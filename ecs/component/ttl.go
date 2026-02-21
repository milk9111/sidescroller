package component

// TTL is a simple frame-based time-to-live component. Systems may add this
// component to an entity to have it automatically destroyed after the given
// number of update ticks.
type TTL struct {
	// Frames remaining for the TTL (in update ticks)
	Frames int
}

var TTLComponent = NewComponent[TTL]()
