package system

import (
	"fmt"
	"math"

	"gopkg.in/yaml.v3"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

type Action func(ctx *AIActionContext)

type AIActionContext struct {
	World           *ecs.World
	Entity          ecs.Entity
	AI              *component.AI
	State           *component.AIState
	Context         *component.AIContext
	Config          *component.AIConfig
	PlayerFound     bool
	PlayerX         float64
	PlayerY         float64
	GetPosition     func() (x, y float64)
	GetVelocity     func() (x, y float64)
	SetVelocity     func(x, y float64)
	ChangeAnimation func(name string)
	FacingLeft      func(facingLeft bool)
	EnqueueEvent    func(ev component.EventID)
}

type StateDef struct {
	OnEnter []Action
	While   []Action
	OnExit  []Action
}

type FSMDef struct {
	Initial     component.StateID
	States      map[component.StateID]StateDef
	Transitions map[component.StateID]map[component.EventID]component.StateID
	Checkers    []TransitionCheckerDef
}

type RawFSM struct {
	Initial string              `yaml:"initial"`
	States  map[string]RawState `yaml:"states"`
	// Transitions can be either the old-style map[from]map[event]to
	// or the new-style map[from][]map[condition]value where condition
	// names may be looked up in the transition registry.
	Transitions map[string]any `yaml:"transitions"`
}

type RawState struct {
	OnEnter []map[string]any `yaml:"on_enter"`
	While   []map[string]any `yaml:"while"`
	OnExit  []map[string]any `yaml:"on_exit"`
}

var actionRegistry = map[string]func(any) Action{
	"print": func(arg any) Action {
		msg := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			fmt.Println("ai:", msg)
		}
	},
	"set_animation": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			if ctx != nil && ctx.ChangeAnimation != nil {
				ctx.ChangeAnimation(name)
			}
		}
	},
	"stop_x": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.GetVelocity == nil || ctx.SetVelocity == nil {
				return
			}
			_, y := ctx.GetVelocity()
			ctx.SetVelocity(0, y)
		}
	},
	"move_towards_player": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.AI == nil || !ctx.PlayerFound || ctx.GetPosition == nil || ctx.GetVelocity == nil || ctx.SetVelocity == nil {
				return
			}
			ex, _ := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			dir := 0.0
			if math.Abs(dx) > 0.001 {
				if dx > 0 {
					dir = 1
				} else {
					dir = -1
				}
			}
			_, y := ctx.GetVelocity()
			ctx.SetVelocity(dir*ctx.AI.MoveSpeed, y)
		}
	},
	"face_player": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || !ctx.PlayerFound || ctx.GetPosition == nil || ctx.FacingLeft == nil {
				return
			}
			ex, _ := ctx.GetPosition()
			ctx.FacingLeft(ctx.PlayerX < ex)
		}
	},
	"start_timer": func(arg any) Action {
		seconds := asFloat(arg)
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.Context == nil {
				return
			}
			ctx.Context.Timer = seconds
		}
	},
	"start_attack_timer": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.Context == nil || ctx.AI == nil {
				return
			}
			frames := float64(ctx.AI.AttackFrames)
			if frames <= 0 {
				frames = 20
			}
			// store timer in seconds (frames / TPS) to match tick_timer which
			// decrements by 1/ebiten.ActualTPS() each update
			tps := ebiten.ActualTPS()
			if tps <= 0 {
				ctx.Context.Timer = frames
			} else {
				ctx.Context.Timer = frames / tps
			}
		}
	},
	"tick_timer": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.Context == nil || ctx.EnqueueEvent == nil {
				return
			}
			ctx.Context.Timer -= 1 / ebiten.ActualTPS()
			if ctx.Context.Timer <= 0 {
				ctx.EnqueueEvent(component.EventID("timer_expired"))
			}
		}
	},
	"emit_event": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.EnqueueEvent == nil {
				return
			}
			ctx.EnqueueEvent(component.EventID(name))
		}
	},
	"add_white_flash": func(arg any) Action {
		// arg may be a number (frames) or map; we accept numeric frames and use a default interval
		frames := 30
		if arg != nil {
			switch v := arg.(type) {
			case int:
				frames = v
			case float64:
				frames = int(v)
			}
		}
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.WhiteFlashComponent, component.WhiteFlash{Frames: frames, Interval: 5, Timer: 0, On: true})
		}
	},
	"add_invulnerable": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.InvulnerableComponent, component.Invulnerable{})
		}
	},
	"remove_invulnerable": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Remove(ctx.World, ctx.Entity, component.InvulnerableComponent)
		}
	},
}

type TransitionChecker func(ctx *AIActionContext) bool

type TransitionCheckerDef struct {
	From  component.StateID
	Event component.EventID
	Check TransitionChecker
}

var transitionRegistry = map[string]func(any) TransitionChecker{
	"always": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool { return true }
	},
	"sees_player": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil || ctx.GetPosition == nil {
				return false
			}
			if !ctx.PlayerFound {
				return false
			}
			if ctx.AI.FollowRange <= 0 {
				return false
			}
			ex, _ := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			return math.Hypot(dx, 0) <= ctx.AI.FollowRange
		}
	},
	"loses_player": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil || ctx.GetPosition == nil {
				return false
			}
			// if player not found, it's a loss
			if !ctx.PlayerFound {
				return true
			}
			if ctx.AI.FollowRange <= 0 {
				return false
			}
			ex, _ := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			return math.Hypot(dx, 0) > ctx.AI.FollowRange
		}
	},
	"timer_expired": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.Context == nil {
				return false
			}
			res := ctx.Context.Timer <= 0
			return res
		}
	},
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case float64:
		return t
	case float32:
		return float64(t)
	default:
		return 0
	}
}

func CompileFSM(raw RawFSM) (*FSMDef, error) {
	if raw.Initial == "" {
		return nil, fmt.Errorf("fsm: missing initial state")
	}

	states := map[component.StateID]StateDef{}
	build := func(list []map[string]any) ([]Action, error) {
		if len(list) == 0 {
			return nil, nil
		}
		out := make([]Action, 0, len(list))
		for _, e := range list {
			for k, v := range e {
				makeAction, ok := actionRegistry[k]
				if !ok {
					return nil, fmt.Errorf("fsm: unknown action %q", k)
				}
				out = append(out, makeAction(v))
			}
		}
		return out, nil
	}

	for name, s := range raw.States {
		onEnter, err := build(s.OnEnter)
		if err != nil {
			return nil, err
		}
		while, err := build(s.While)
		if err != nil {
			return nil, err
		}
		onExit, err := build(s.OnExit)
		if err != nil {
			return nil, err
		}
		states[component.StateID(name)] = StateDef{
			OnEnter: onEnter,
			While:   while,
			OnExit:  onExit,
		}
	}

	transitions := map[component.StateID]map[component.EventID]component.StateID{}
	var checkers []TransitionCheckerDef

	for from, rawVal := range raw.Transitions {
		fromID := component.StateID(from)
		transitions[fromID] = map[component.EventID]component.StateID{}

		switch v := rawVal.(type) {
		case map[string]any:
			for evName, toVal := range v {
				// simple mapping: event -> state
				if toStr, ok := toVal.(string); ok {
					transitions[fromID][component.EventID(evName)] = component.StateID(toStr)
					continue
				}
				// registry-driven transition: evName is a condition name
				if maker, ok := transitionRegistry[evName]; ok {
					var toState string
					var arg any
					if m, ok := toVal.(map[string]any); ok {
						if ts, ok2 := m["to"].(string); ok2 {
							toState = ts
						}
						arg = m["arg"]
					} else if s, ok2 := toVal.(string); ok2 {
						toState = s
					}
					if toState == "" {
						return nil, fmt.Errorf("fsm: missing to state for transition %s.%s", from, evName)
					}
					eid := component.EventID(fmt.Sprintf("__cond_%s_%s", from, evName))
					transitions[fromID][eid] = component.StateID(toState)
					checkers = append(checkers, TransitionCheckerDef{From: fromID, Event: eid, Check: maker(arg)})
					continue
				}
				return nil, fmt.Errorf("fsm: invalid transition value for %s.%s", from, evName)
			}
		case []any:
			for i, item := range v {
				m, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("fsm: invalid transition entry %v", item)
				}
				for key, val := range m {
					if maker, ok := transitionRegistry[key]; ok {
						var toState string
						var arg any
						if mv, ok2 := val.(map[string]any); ok2 {
							if ts, ok3 := mv["to"].(string); ok3 {
								toState = ts
							}
							arg = mv["arg"]
						} else if s, ok3 := val.(string); ok3 {
							toState = s
						}
						if toState == "" {
							return nil, fmt.Errorf("fsm: missing to state for transition %s", key)
						}
						eid := component.EventID(fmt.Sprintf("__cond_%s_%d", from, i))
						transitions[fromID][eid] = component.StateID(toState)
						checkers = append(checkers, TransitionCheckerDef{From: fromID, Event: eid, Check: maker(arg)})
					} else {
						if toState, ok2 := val.(string); ok2 {
							transitions[fromID][component.EventID(key)] = component.StateID(toState)
						} else {
							return nil, fmt.Errorf("fsm: invalid transition mapping for %s -> %v", key, val)
						}
					}
				}
			}
		default:
			return nil, fmt.Errorf("fsm: invalid transitions type for state %s", from)
		}
	}

	return &FSMDef{
		Initial:     component.StateID(raw.Initial),
		States:      states,
		Transitions: transitions,
		Checkers:    checkers,
	}, nil
}

func LoadFSMFromPrefab(path string) (*FSMDef, error) {
	data, err := prefabs.Load(path)
	if err != nil {
		return nil, err
	}
	var raw RawFSM
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return CompileFSM(raw)
}

func DefaultEnemyFSM() *FSMDef {
	f := &FSMDef{
		Initial: component.StateID("idle"),
		States: map[component.StateID]StateDef{
			component.StateID("idle"): {
				OnEnter: []Action{actionRegistry["set_animation"]("idle")},
				While:   []Action{actionRegistry["stop_x"](nil)},
			},
			component.StateID("follow"): {
				OnEnter: []Action{actionRegistry["set_animation"]("run")},
				While: []Action{
					actionRegistry["move_towards_player"](nil),
					actionRegistry["face_player"](nil),
				},
			},
			component.StateID("attack"): {
				OnEnter: []Action{
					actionRegistry["set_animation"]("attack"),
					actionRegistry["start_attack_timer"](nil),
				},
				While: []Action{
					actionRegistry["stop_x"](nil),
					actionRegistry["tick_timer"](nil),
				},
			},
		},
		Transitions: map[component.StateID]map[component.EventID]component.StateID{
			component.StateID("idle"): {
				component.EventID("sees_player"): component.StateID("follow"),
			},
			component.StateID("follow"): {
				component.EventID("loses_player"): component.StateID("idle"),
			},
			component.StateID("attack"): {
				component.EventID("timer_expired"): component.StateID("follow"),
				component.EventID("loses_player"):  component.StateID("idle"),
			},
		},
	}

	return f
}

func CompileFSMSpec(spec prefabs.FSMSpec) (*FSMDef, error) {
	raw := RawFSM{
		Initial:     spec.Initial,
		States:      map[string]RawState{},
		Transitions: map[string]any{},
	}
	// copy transitions into the flexible raw.Transitions shape
	for from, evs := range spec.Transitions {
		m := map[string]any{}
		for ev, to := range evs {
			m[ev] = to
		}
		raw.Transitions[from] = m
	}
	for name, s := range spec.States {
		raw.States[name] = RawState{
			OnEnter: s.OnEnter,
			While:   s.While,
			OnExit:  s.OnExit,
		}
	}
	return CompileFSM(raw)
}
