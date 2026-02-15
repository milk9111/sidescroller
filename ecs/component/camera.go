package component

type Camera struct {
	TargetName string
	Zoom       float64
	// Smoothness controls how quickly the camera follows the target.
	// Range [0,1]. 1.0 = instant follow, lower values = smoother/lagging follow.
	Smoothness float64
	// LookOffset controls how many world units the camera will shift when
	// the player holds the look up/down input. Positive values move the
	// camera in the input direction (down positive).
	LookOffset float64
	// LookSmooth is an optional smoothing factor applied to the look offset
	// applied each frame. Range [0,1].
	LookSmooth float64
}

var CameraComponent = NewComponent[Camera]()
