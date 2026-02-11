package component

import "github.com/milk9111/sidescroller/prefabs"

// StateID identifies an AI FSM state.
type StateID string

// EventID identifies an AI FSM event.
type EventID string

const DefaultAIFSMName = "enemy_default"

// AIState stores the current FSM state.
type AIState struct {
	Current StateID
}

// AIContext stores per-entity AI runtime data.
type AIContext struct {
	Timer float64
}

// AIConfig stores either a reference to an FSM by name or an embedded FSMSpec.
type AIConfig struct {
	FSM  string
	Spec *prefabs.FSMSpec
}

var AIStateComponent = NewComponent[AIState]()
var AIContextComponent = NewComponent[AIContext]()
var AIConfigComponent = NewComponent[AIConfig]()
