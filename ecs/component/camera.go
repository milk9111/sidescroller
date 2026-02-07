package component

type Camera struct {
	TargetName string
}

var CameraComponent = NewComponent[Camera]()
