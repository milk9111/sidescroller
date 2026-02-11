package component

// Hurtbox represents a defensive AABB relative to the entity transform.
type Hurtbox struct {
	Width   float64
	Height  float64
	OffsetX float64
	OffsetY float64
}

var HurtboxComponent = NewComponent[[]Hurtbox]()
