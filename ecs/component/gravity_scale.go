package component

// GravityScale scales world gravity for a dynamic physics body.
// 1.0 = normal gravity, 0.0 = no gravity.
type GravityScale struct {
	Scale float64
}

var GravityScaleComponent = NewComponent[GravityScale]()
