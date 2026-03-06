package component

type Color struct {
	R float64
	G float64
	B float64
	A float64
}

var ColorComponent = NewComponent[Color]()
