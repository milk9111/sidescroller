package components

// CameraFollow stores camera targeting data.
type CameraFollow struct {
	TargetEntity int
	OffsetX      float64
	OffsetY      float64
}

// CameraState stores camera view values for reference/debug.
type CameraState struct {
	PosX float64
	PosY float64
	Zoom float64
}
