package component

type LeverState string

const (
	LeverStateOpen    LeverState = "open"
	LeverStateClosing LeverState = "closing"
	LeverStateClosed  LeverState = "closed"
)

type Lever struct {
	OpenAnimation    string
	ClosingAnimation string
	ClosedAnimation  string
	State            LeverState
}

var LeverComponent = NewComponent[Lever]()
