package component

type Transform struct {
	X        float64
	Y        float64
	ScaleX   float64
	ScaleY   float64
	Rotation float64
}

var TransformComponent = NewComponent[Transform]()
