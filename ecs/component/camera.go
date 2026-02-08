package component

type Camera struct {
	TargetName string
	Zoom       float64
}

var CameraComponent = NewComponent[Camera]()
