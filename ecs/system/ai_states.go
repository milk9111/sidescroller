package system

import (
	"fmt"
	"math"

	"gopkg.in/yaml.v3"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

type Action func(ctx *AIActionContext, dt float64)

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
}

type RawFSM struct {
	Initial     string                       `yaml:"initial"`
	States      map[string]RawState          `yaml:"states"`
	Transitions map[string]map[string]string `yaml:"transitions"`
}

type RawState struct {
	OnEnter []map[string]any `yaml:"on_enter"`
	While   []map[string]any `yaml:"while"`
	OnExit  []map[string]any `yaml:"on_exit"`
}

var actionRegistry = map[string]func(any) Action{
	"print": func(arg any) Action {
		msg := fmt.Sprint(arg)
		return func(ctx *AIActionContext, _ float64) {
			fmt.Println("ai:", msg)
		}
	},
	"set_animation": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext, _ float64) {
			if ctx != nil && ctx.ChangeAnimation != nil {
				ctx.ChangeAnimation(name)
			}
		}
	},
	"stop_x": func(_ any) Action {
		return func(ctx *AIActionContext, _ float64) {
			if ctx == nil || ctx.GetVelocity == nil || ctx.SetVelocity == nil {
				return
			}
			_, y := ctx.GetVelocity()
			ctx.SetVelocity(0, y)
		}
	},
	"move_towards_player": func(_ any) Action {
		return func(ctx *AIActionContext, _ float64) {
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
		return func(ctx *AIActionContext, _ float64) {
			if ctx == nil || !ctx.PlayerFound || ctx.GetPosition == nil || ctx.FacingLeft == nil {
				return
			}
			ex, _ := ctx.GetPosition()
			ctx.FacingLeft(ctx.PlayerX < ex)
		}
	},
	"start_timer": func(arg any) Action {
		seconds := asFloat(arg)
		return func(ctx *AIActionContext, _ float64) {
			if ctx == nil || ctx.Context == nil {
				return
			}
			ctx.Context.Timer = seconds
		}
	},
	"start_attack_timer": func(_ any) Action {
		return func(ctx *AIActionContext, _ float64) {
			if ctx == nil || ctx.Context == nil || ctx.AI == nil {
				return
			}
			frames := float64(ctx.AI.AttackFrames)
			if frames <= 0 {
				frames = 20
			}
			ctx.Context.Timer = frames
		}
	},
	"tick_timer": func(_ any) Action {
		return func(ctx *AIActionContext, dt float64) {
			if ctx == nil || ctx.Context == nil || ctx.EnqueueEvent == nil {
				return
			}
			ctx.Context.Timer -= dt
			if ctx.Context.Timer <= 0 {
				ctx.EnqueueEvent(component.EventID("timer_expired"))
			}
		}
	},
	"emit_event": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext, _ float64) {
			if ctx == nil || ctx.EnqueueEvent == nil {
				return
			}
			ctx.EnqueueEvent(component.EventID(name))
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
	for from, evs := range raw.Transitions {
		fromID := component.StateID(from)
		transitions[fromID] = map[component.EventID]component.StateID{}
		for ev, to := range evs {
			transitions[fromID][component.EventID(ev)] = component.StateID(to)
		}
	}

	return &FSMDef{
		Initial:     component.StateID(raw.Initial),
		States:      states,
		Transitions: transitions,
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
	return &FSMDef{
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
				component.EventID("see_player"): component.StateID("follow"),
			},
			component.StateID("follow"): {
				component.EventID("lose_player"):     component.StateID("idle"),
				component.EventID("in_attack_range"): component.StateID("attack"),
			},
			component.StateID("attack"): {
				component.EventID("timer_expired"): component.StateID("follow"),
				component.EventID("lose_player"):   component.StateID("idle"),
			},
		},
	}
}
