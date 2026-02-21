package component

// Hitbox represents an offensive AABB relative to the entity transform.
type Hitbox struct {
	Width      float64
	Height     float64
	OffsetX    float64
	OffsetY    float64
	Damage     int
	Anim       string
	Frames     []int
	HitTargets map[uint64]bool
}

var HitboxComponent = NewComponent[[]Hitbox]()
