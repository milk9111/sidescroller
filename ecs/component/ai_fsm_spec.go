package component

// AIFSMSpec is a YAML-agnostic representation of an AI finite state machine
// spec used by runtime systems.
type AIFSMSpec struct {
	ScriptPath      string
	ScriptLifecycle bool
	Initial         string
	States          map[string]AIFSMStateSpec
	Transitions     map[string][]map[string]any
}

type AIFSMStateSpec struct {
	OnEnter []map[string]any
	While   []map[string]any
	OnExit  []map[string]any
}
