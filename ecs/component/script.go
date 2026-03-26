package component

type Script struct {
	Path    string
	Paths   []string
	Modules []string
}

var ScriptComponent = NewComponent[Script]()

type ScriptRuntime struct {
	Started bool
}

var ScriptRuntimeComponent = NewComponent[ScriptRuntime]()

type ScriptSignalEvent struct {
	Name             string
	SourceGameEntity string
	ExcludedEntity   uint64
	HasPosition      bool
	PositionX        float64
	PositionY        float64
}

type ScriptSignalQueue struct {
	Events []ScriptSignalEvent
}

var ScriptSignalQueueComponent = NewComponent[ScriptSignalQueue]()

type GlobalHitSignalQueue struct {
	Events []ScriptSignalEvent
}

var GlobalHitSignalQueueComponent = NewComponent[GlobalHitSignalQueue]()

// ScriptState holds a string representation of the script-managed state
// (for example `state["current_state"]` in Tengo scripts).
type ScriptState struct {
	Current string
}

var ScriptStateComponent = NewComponent[ScriptState]()
