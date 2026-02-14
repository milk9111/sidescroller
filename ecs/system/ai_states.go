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
			ex, ey := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			dy := ctx.PlayerY - ey

			// Stop slightly before nominal attack range so we don't overshoot the
			// target before the FSM transitions into attack.
			stopDistance := ctx.AI.AttackRange + 24
			if stopDistance < 24 {
				stopDistance = 24
			}

			// Reduce jitter when vertically stacked with the player by using a
			// wider horizontal deadzone.
			horizontalDeadzone := 4.0
			if math.Abs(dy) > 24 {
				horizontalDeadzone = 10
			}

			dir := 0.0
			if math.Abs(dx) > horizontalDeadzone && math.Abs(dx) > stopDistance {
				if dx > 0 {
					dir = 1
				} else {
					dir = -1
				}
			}

			// Add simple local separation so enemies spread instead of clustering.
			if ctx.World != nil {
				const desiredSeparation = 36.0
				const verticalNeighborBand = 28.0
				repel := 0.0

				ecs.ForEach2(ctx.World,
					component.AITagComponent.Kind(),
					component.TransformComponent.Kind(),
					func(other ecs.Entity, _ *component.AITag, ot *component.Transform) {
						if other == ctx.Entity || ot == nil {
							return
						}
						if math.Abs(ot.Y-ey) > verticalNeighborBand {
							return
						}
						deltaX := ex - ot.X
						distX := math.Abs(deltaX)
						if distX < 0.001 || distX >= desiredSeparation {
							return
						}
						strength := (desiredSeparation - distX) / desiredSeparation
						if deltaX > 0 {
							repel += strength
						} else {
							repel -= strength
						}
					},
				)

				if repel > 1 {
					repel = 1
				} else if repel < -1 {
					repel = -1
				}

				dir += repel * 0.9
				if dir > 1 {
					dir = 1
				} else if dir < -1 {
					dir = -1
				}
				if math.Abs(dir) < 0.2 {
					dir = 0
				}
			}

			// If moving horizontally, check for ground ahead on the current platform.
			// If no ground is found, stop moving to avoid falling off edges.
			// Consult precomputed navigation data set by the AINavigationSystem.
			if ctx.World != nil && dir != 0 {
				if nav, ok := ecs.Get(ctx.World, ctx.Entity, component.AINavigationComponent.Kind()); ok && nav != nil {
					if dir > 0 && !nav.GroundAheadRight {
						dir = 0
					} else if dir < 0 && !nav.GroundAheadLeft {
						dir = 0
					}
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
			fmt.Println("starting timer for", seconds, "seconds")
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
			_ = ecs.Add(ctx.World, ctx.Entity, component.WhiteFlashComponent.Kind(), &component.WhiteFlash{Frames: frames, Interval: 5, Timer: 0, On: true})
		}
	},
	"add_invulnerable": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.InvulnerableComponent.Kind(), &component.Invulnerable{})
		}
	},
	"remove_invulnerable": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Remove(ctx.World, ctx.Entity, component.InvulnerableComponent.Kind())
		}
	},
	"destroy_self": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}

			fmt.Println("destroying entity", ctx.Entity)
			if ok := ecs.DestroyEntity(ctx.World, ctx.Entity); !ok {
				panic(fmt.Sprintf("ai: failed to destroy entity %d", ctx.Entity))
			}
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
			dx, dy := ctx.PlayerX-ex, ctx.PlayerY-0
			// Prefer using the full 2D distance between AI and player.
			// getPosition returns the AI's position; use PlayerY from context.
			ex2, ey2 := ctx.GetPosition()
			dx = ctx.PlayerX - ex2
			dy = ctx.PlayerY - ey2
			return math.Hypot(dx, dy) <= ctx.AI.FollowRange
		}
	},
	"sees_player_and_not_reached_edge": func(arg any) TransitionChecker {
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

			nav, ok := ecs.Get(ctx.World, ctx.Entity, component.AINavigationComponent.Kind())
			if !ok || nav == nil {
				// no nav info: fall back to simple sees_player
				return true
			}

			// Determine facing from the sprite when available; default to facing
			// toward the player if sprite not present.
			facingLeft := false
			if sp, ok := ecs.Get(ctx.World, ctx.Entity, component.SpriteComponent.Kind()); ok && sp != nil {
				facingLeft = sp.FacingLeft
			} else {
				// fallback: if player is left of AI, consider facing left
				ex2, _ := ctx.GetPosition()
				if ctx.PlayerX < ex2 {
					facingLeft = true
				}
			}

			// If there's ground ahead in the facing direction, allow follow.
			frontHasGround := true
			if facingLeft {
				frontHasGround = nav.GroundAheadLeft
			} else {
				frontHasGround = nav.GroundAheadRight
			}

			// If front is safe, allow follow.
			if frontHasGround {
				// continue to distance check below
			} else {
				// Allow turning toward the player if player is nearly level
				// with the AI (so they can see/face the player). Use the same
				// vertical tolerance as other AI code (24 px).
				_, ey2 := ctx.GetPosition()
				dy := math.Abs(ctx.PlayerY - ey2)
				if dy <= 24 {
					// treat as safe to follow/face
				} else {
					return false
				}
			}

			ex, _ := ctx.GetPosition()
			dx, dy := ctx.PlayerX-ex, ctx.PlayerY-0
			// Prefer using the full 2D distance between AI and player.
			// getPosition returns the AI's position; use PlayerY from context.
			ex2, ey2 := ctx.GetPosition()
			dx = ctx.PlayerX - ex2
			dy = ctx.PlayerY - ey2
			return math.Hypot(dx, dy) <= ctx.AI.FollowRange
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
			dx, dy := ctx.PlayerX-ex, ctx.PlayerY-0
			ex2, ey2 := ctx.GetPosition()
			dx = ctx.PlayerX - ex2
			dy = ctx.PlayerY - ey2
			return math.Hypot(dx, dy) > ctx.AI.FollowRange
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
	"out_of_health": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil {
				return false
			}
			health, _ := ecs.Get(ctx.World, ctx.Entity, component.HealthComponent.Kind())
			return health.Current <= 0
		}
	},
	"reached_edge": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			nav, ok := ecs.Get(ctx.World, ctx.Entity, component.AINavigationComponent.Kind())
			if !ok || nav == nil {
				return false
			}

			return !nav.GroundAheadLeft || !nav.GroundAheadRight
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
	// spec.Transitions is an ordered slice of maps (to preserve priority),
	// so convert each entry into a []any of map[string]any to feed CompileFSM.
	for from, evs := range spec.Transitions {
		items := make([]any, 0, len(evs))
		for _, evmap := range evs {
			m := map[string]any{}
			for k, v := range evmap {
				m[k] = v
			}
			items = append(items, m)
		}
		raw.Transitions[from] = items
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
