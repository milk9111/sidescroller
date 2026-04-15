package component

type MovingPlatformMode string

const (
	MovingPlatformModeContinuous MovingPlatformMode = "continuous"
	MovingPlatformModeTouch      MovingPlatformMode = "touch"
)

type MovingPlatform struct {
	Mode             MovingPlatformMode
	Speed            float64
	StartX           float64
	StartY           float64
	DestX            float64
	DestY            float64
	WaitFrames       int
	Progress         float64
	Direction        float64
	DeltaX           float64
	DeltaY           float64
	Moving           bool
	TouchedWhileIdle bool
	Initialized      bool
	StartAtTarget    bool
	WasTouching      bool
	FramesUntilMove  int
}

var MovingPlatformComponent = NewComponent[MovingPlatform]()
