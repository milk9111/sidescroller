package component

// AIPhase defines phase-gated transition overrides for an AI FSM.
type AIPhase struct {
	Name                string
	StartWhen           []map[string]any
	TransitionOverrides map[string][]map[string]any
	OnEnter             []map[string]any
}

// AIPhaseController stores authored phase data and the base transitions.
type AIPhaseController struct {
	BaseTransitions         map[string][]map[string]any
	Phases                  []AIPhase
	ResetStateOnPhaseChange bool
}

// AIPhaseRuntime stores runtime progression through phases.
type AIPhaseRuntime struct {
	Initialized  bool
	CurrentPhase int
}

var AIPhaseControllerComponent = NewComponent[AIPhaseController]()
var AIPhaseRuntimeComponent = NewComponent[AIPhaseRuntime]()
