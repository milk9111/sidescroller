package component

// Boss stores data-driven phase and attack-pattern configuration for a boss entity.
type Boss struct {
	DisplayName string
	Phases      []BossPhase
}

type BossPhase struct {
	Name        string
	HPTrigger   int
	PatternMode string
	OnEnter     []map[string]any
	Arena       []map[string]any
	Patterns    []BossAttackPattern
}

type BossAttackPattern struct {
	Name           string
	CooldownFrames int
	Actions        []map[string]any
}

// BossRuntime stores runtime-only state for phase progression and pattern selection.
type BossRuntime struct {
	Initialized  bool
	CurrentPhase int
	PatternIndex int
	Cooldown     int
	// PendingDelays holds scheduled action sequences that execute after a
	// frame delay. Used by BossSystem to coordinate timed OnEnter sequences.
	PendingDelays []DelayedAction
}

type DelayedAction struct {
	Frames  int
	Actions []map[string]any
}

var BossComponent = NewComponent[Boss]()
var BossRuntimeComponent = NewComponent[BossRuntime]()
