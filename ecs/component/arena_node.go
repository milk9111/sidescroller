package component

// ArenaNode marks an entity as part of a named arena control group.
// AI actions can toggle these values by group name.
type ArenaNode struct {
	Group             string
	Active            bool
	HazardEnabled     bool
	TransitionEnabled bool
}

// ArenaNodeRuntime caches authored templates that can be restored when
// arena toggles are re-enabled.
type ArenaNodeRuntime struct {
	Initialized           bool
	HasHazardTemplate     bool
	HazardTemplate        Hazard
	HasTransitionTemplate bool
	TransitionTemplate    Transition
}

var ArenaNodeComponent = NewComponent[ArenaNode]()
var ArenaNodeRuntimeComponent = NewComponent[ArenaNodeRuntime]()
