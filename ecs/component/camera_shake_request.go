package component

// CameraShakeRequest asks the camera system to apply a short shake effect.
// Intensity is measured in world units (pixels at zoom=1).
type CameraShakeRequest struct {
	Frames    int
	Intensity float64
}

var CameraShakeRequestComponent = NewComponent[CameraShakeRequest]()
