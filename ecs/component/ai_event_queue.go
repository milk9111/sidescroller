package component

// AIEventQueue is a one-tick queue of FSM events for AISystem to consume.
// Producer systems append events here; AISystem drains and removes it each tick.
type AIEventQueue struct {
	Events []string
}

var AIEventQueueComponent = NewComponent[AIEventQueue]()
