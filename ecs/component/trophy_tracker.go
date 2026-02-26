package component

type TrophyTracker struct {
	Count int
}

var TrophyTrackerComponent = NewComponent[TrophyTracker]()
