package component

// TransitionRuntime holds transient state for an in-progress level transition.
type TransitionPhase int

const (
	TransitionNone TransitionPhase = iota
	TransitionFadeOut
	TransitionFadeIn
)

type TransitionRuntime struct {
	Phase   TransitionPhase
	Alpha   float64
	Timer   int
	Req     LevelChangeRequest
	ReqSent bool
}

var TransitionRuntimeComponent = NewComponent[TransitionRuntime]()
