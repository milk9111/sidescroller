package component

// CollisionLayer allows entities to declare a collision category and mask
// so the physics system can selectively enable/disable collisions between
// groups of objects.
type CollisionLayer struct {
	// Category is a bitmask of this entity's collision category. If zero,
	// the physics system will treat it as category 1.
	Category uint32 `json:"category,omitempty"`
	// Mask is a bitmask of categories this entity should collide with. If
	// zero, the physics system will treat it as all-bits set (collide with all).
	Mask uint32 `json:"mask,omitempty"`
}

var CollisionLayerComponent = NewComponent[CollisionLayer]()
