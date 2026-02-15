package component

// Hazard marks an entity as dangerous on overlap.
// Bounds are expressed in world units relative to Transform (top-left origin).
type Hazard struct {
	Width   float64
	Height  float64
	OffsetX float64
	OffsetY float64
}

var HazardComponent = NewComponent[Hazard]()
