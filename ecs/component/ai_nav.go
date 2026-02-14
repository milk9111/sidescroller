package component

// AINavigation stores precomputed ground-ahead checks for AI entities.
type AINavigation struct {
	GroundAheadLeft  bool
	GroundAheadRight bool
}

var AINavigationComponent = NewComponent[AINavigation]()
