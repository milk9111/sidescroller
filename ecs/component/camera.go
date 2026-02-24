package component

type Camera struct {
	TargetName  string
	Zoom        float64
	Smoothness  float64
	LookOffset  float64
	LookSmooth  float64
	LockEnabled bool
	LockCapture bool
	LockCenterX float64
	LockCenterY float64
}

var CameraComponent = NewComponent[Camera]()
