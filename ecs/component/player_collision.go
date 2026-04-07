package component

// PlayerCollision stores per-player collision state derived from physics contacts.
type PlayerCollision struct {
	Grounded    bool
	GroundGrace int
	// Wall: 0 = none, 1 = left, 2 = right
	Wall int
	// Clamber is true when physics detected a ledge the player can mantle onto.
	Clamber bool
	// ClamberTargetX/Y is the body center to move toward while clambering.
	ClamberTargetX float64
	ClamberTargetY float64
	// CollidedAI is true when the player has overlapped an AI/enemy this step
	CollidedAI bool
}

var PlayerCollisionComponent = NewComponent[PlayerCollision]()
