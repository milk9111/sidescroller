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
}

var LevelChangeRequestComponent = NewComponent[LevelChangeRequest]()
