package component

type Parallax struct {
	FactorX float64
	FactorY float64

	AnchorCameraX    float64
	AnchorCameraY    float64
	HasAnchorCameraX bool
	HasAnchorCameraY bool

	BaseX float64
	BaseY float64

	CameraBaseX float64
	CameraBaseY float64

	Initialized bool
}

var ParallaxComponent = NewComponent[Parallax]()
