package component

// LevelLoaded is a transient marker added by the outer Game loop to tell the
// TransitionSystem that the level load + player placement has completed and
// the system can begin fading back in.
type LevelLoaded struct{}

var LevelLoadedComponent = NewComponent[LevelLoaded]()
