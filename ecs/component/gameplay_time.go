package component

// GameplayTime stores the current gameplay simulation scale for the world.
type GameplayTime struct {
	Scale float64
}

var GameplayTimeComponent = NewComponent[GameplayTime]()
