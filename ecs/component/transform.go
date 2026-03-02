package component

type Transform struct {
	X        float64
	Y        float64
	ScaleX   float64
	ScaleY   float64
	Rotation float64
	Parent   uint64

	WorldX        float64
	WorldY        float64
	WorldScaleX   float64
	WorldScaleY   float64
	WorldRotation float64
}

var TransformComponent = NewComponent[Transform]()
