package component

// Anchor describes a spawned anchor moving to a target point.
type Anchor struct {
	TargetX float64
	TargetY float64
	Speed   float64 // pixels per update
}

var AnchorComponent = NewComponent[Anchor]()
