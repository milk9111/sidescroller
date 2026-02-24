package component

// Pickup is a generic collectible with reusable bob/collision behavior.
type Pickup struct {
	Kind            string
	BaseY           float64
	BobAmplitude    float64
	BobSpeed        float64
	BobPhase        float64
	CollisionWidth  float64
	CollisionHeight float64
	GrantDoubleJump bool
	GrantWallGrab   bool
	GrantAnchor     bool
	Initialized     bool
}

var PickupComponent = NewComponent[Pickup]()
