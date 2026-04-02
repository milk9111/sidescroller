package component

// DebugMessage stores the current top-of-screen debug/tutorial message state.
type DebugMessage struct {
	Width           int
	Height          int
	Message         string
	RemainingFrames int
}

var DebugMessageComponent = NewComponent[DebugMessage]()
