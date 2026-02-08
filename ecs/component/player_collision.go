package component

// PlayerCollision stores per-player collision state derived from physics contacts.
type PlayerCollision struct {
	Grounded    bool
	GroundGrace int
	// Wall: 0 = none, 1 = left, 2 = right
	Wall int
}

var PlayerCollisionComponent = NewComponent[PlayerCollision]()
