package component

// AIStateInterrupt is a one-shot event to request an AI FSM event be enqueued
// for the AISystem. Systems can add this component to an AI entity and the
// AISystem will consume it during its update.
type AIStateInterrupt struct {
	Event string
}

var AIStateInterruptComponent = NewComponent[AIStateInterrupt]()
