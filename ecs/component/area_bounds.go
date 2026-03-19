package component

// AreaBounds stores an axis-aligned authored rectangle relative to the owning
// entity transform. It is shared by editor box-placement and runtime overlap/
// stamping logic for entities that span multiple tiles.
type AreaBounds struct {
	Bounds AABB
}

var AreaBoundsComponent = NewComponent[AreaBounds]()
