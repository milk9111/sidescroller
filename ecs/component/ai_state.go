package component

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

// AIConfig stores the FSM configuration reference for an entity.
type AIConfig struct {
	FSM string
}

var AIStateComponent = NewComponent[AIState]()
var AIContextComponent = NewComponent[AIContext]()
var AIConfigComponent = NewComponent[AIConfig]()
