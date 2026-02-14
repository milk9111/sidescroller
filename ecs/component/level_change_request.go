package component

// LevelChangeRequest is a one-shot request emitted by gameplay systems (e.g.
// TransitionSystem) to ask the outer game loop to load a different level.
//
// This keeps systems independent: systems only emit data; the Game loop owns
// IO/world reinitialization.
type LevelChangeRequest struct {
	TargetLevel       string
	SpawnTransitionID string
	EnterDir          TransitionDirection
	FromTransitionID  string
	FromTransitionEnt uint64 // optional debug field (ecs.Entity is uint64)
	// FromFacingLeft records the player's facing when the transition was
	// initiated so the spawn side can match and apply effects (e.g. pop).
	FromFacingLeft bool
	// EntryFromBelow is true when the player was positioned below the
	// transition area at the time of triggering, indicating an upward
	// entry and a candidate for the pop impulse after spawn.
	EntryFromBelow bool
}

var LevelChangeRequestComponent = NewComponent[LevelChangeRequest]()
