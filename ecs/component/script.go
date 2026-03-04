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
}

type ScriptSignalQueue struct {
	Events []ScriptSignalEvent
}

var ScriptSignalQueueComponent = NewComponent[ScriptSignalQueue]()
