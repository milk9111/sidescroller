package component

type Camera struct {
	TargetName string
	Zoom       float64
	// Smoothness controls how quickly the camera follows the target.
	// Range [0,1]. 1.0 = instant follow, lower values = smoother/lagging follow.
	Smoothness float64
}

var CameraComponent = NewComponent[Camera]()
