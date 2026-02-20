package system

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AIPhaseSystem struct{}

func NewAIPhaseSystem() *AIPhaseSystem { return &AIPhaseSystem{} }

func (s *AIPhaseSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	playerEnt, hasPlayer := ecs.First(w, component.PlayerTagComponent.Kind())
	playerX, playerY := 0.0, 0.0
	if hasPlayer {
		playerX, playerY = phaseEntityPosition(w, playerEnt)
	}

	ecs.ForEach7(w,
		component.AITagComponent.Kind(),
		component.AIPhaseControllerComponent.Kind(),
		component.AIPhaseRuntimeComponent.Kind(),
		component.AIConfigComponent.Kind(),
		component.AIStateComponent.Kind(),
		component.AIContextComponent.Kind(),
		component.AIComponent.Kind(),
		func(e ecs.Entity, _ *component.AITag, controller *component.AIPhaseController, runtime *component.AIPhaseRuntime, cfg *component.AIConfig, state *component.AIState, aiCtx *component.AIContext, ai *component.AI) {
			if ai == nil || controller == nil || runtime == nil || cfg == nil || cfg.Spec == nil || state == nil || aiCtx == nil || len(controller.Phases) == 0 {
				return
			}

			if !runtime.Initialized {
				target := -1
				for idx := 0; idx < len(controller.Phases); idx++ {
					if s.phaseConditionsMet(w, e, state, &controller.Phases[idx], hasPlayer, playerX, playerY) {
						target = idx
						break
					}
				}
				if target < 0 {
					return
				}
				s.activatePhase(w, e, controller, runtime, cfg, state, aiCtx, ai, playerEnt, hasPlayer, playerX, playerY, target)
				return
			}

			if runtime.CurrentPhase < 0 {
				runtime.CurrentPhase = 0
			}
			if runtime.CurrentPhase >= len(controller.Phases) {
				runtime.CurrentPhase = len(controller.Phases) - 1
			}

			for next := runtime.CurrentPhase + 1; next < len(controller.Phases); next++ {
				if !s.phaseConditionsMet(w, e, state, &controller.Phases[next], hasPlayer, playerX, playerY) {
					break
				}
				s.activatePhase(w, e, controller, runtime, cfg, state, aiCtx, ai, playerEnt, hasPlayer, playerX, playerY, next)
			}
		},
	)
}

func (s *AIPhaseSystem) phaseConditionsMet(w *ecs.World, e ecs.Entity, state *component.AIState, phase *component.AIPhase, hasPlayer bool, playerX, playerY float64) bool {
	if phase == nil {
		return false
	}
	if len(phase.StartWhen) == 0 {
		return true
	}

	entityX, entityY := phaseEntityPosition(w, e)

	for _, entry := range phase.StartWhen {
		for key, raw := range entry {
			switch key {
			case "always":
				if v, ok := toBool(raw); !ok || !v {
					return false
				}
			case "hp_lte", "health_lte":
				h, ok := ecs.Get(w, e, component.HealthComponent.Kind())
				if !ok || h == nil {
					return false
				}
				limit, ok := toInt(raw)
				if !ok || h.Current > limit {
					return false
				}
			case "hp_gte", "health_gte":
				h, ok := ecs.Get(w, e, component.HealthComponent.Kind())
				if !ok || h == nil {
					return false
				}
				limit, ok := toInt(raw)
				if !ok || h.Current < limit {
					return false
				}
			case "player_in_range":
				if !hasPlayer {
					return false
				}
				rangeLimit, ok := toFloat(raw)
				if !ok {
					return false
				}
				if math.Hypot(playerX-entityX, playerY-entityY) > rangeLimit {
					return false
				}
			case "player_out_of_range":
				if !hasPlayer {
					return false
				}
				rangeLimit, ok := toFloat(raw)
				if !ok {
					return false
				}
				if math.Hypot(playerX-entityX, playerY-entityY) <= rangeLimit {
					return false
				}
			case "state_is":
				want := fmt.Sprint(raw)
				if string(state.Current) != want {
					return false
				}
			default:
				return false
			}
		}
	}
	return true
}

func (s *AIPhaseSystem) activatePhase(w *ecs.World, e ecs.Entity, controller *component.AIPhaseController, runtime *component.AIPhaseRuntime, cfg *component.AIConfig, state *component.AIState, aiCtx *component.AIContext, ai *component.AI, playerEnt ecs.Entity, hasPlayer bool, playerX, playerY float64, target int) {
	if target < 0 || target >= len(controller.Phases) || cfg.Spec == nil {
		return
	}

	phase := &controller.Phases[target]
	merged := copyTransitions(controller.BaseTransitions)
	for from, entries := range phase.TransitionOverrides {
		copied := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			dup := make(map[string]any, len(entry))
			for k, v := range entry {
				dup[k] = v
			}
			copied = append(copied, dup)
		}
		merged[from] = copied
	}
	cfg.Spec.Transitions = merged
	_ = ecs.Add(w, e, component.AIConfigComponent.Kind(), cfg)

	runtime.Initialized = true
	runtime.CurrentPhase = target
	_ = ecs.Add(w, e, component.AIPhaseRuntimeComponent.Kind(), runtime)

	if controller.ResetStateOnPhaseChange {
		state.Current = ""
	}

	ctx := buildPhaseActionContext(w, e, ai, state, aiCtx, cfg, playerEnt, hasPlayer, playerX, playerY)
	applyInlineActions(phase.OnEnter, ctx)
}

func applyInlineActions(actions []map[string]any, ctx *AIActionContext) {
	for _, entry := range actions {
		for key, arg := range entry {
			makeAction, ok := actionRegistry[key]
			if !ok {
				continue
			}
			act := makeAction(arg)
			if act != nil {
				act(ctx)
			}
		}
	}
}

func buildPhaseActionContext(w *ecs.World, ent ecs.Entity, ai *component.AI, state *component.AIState, aiCtx *component.AIContext, cfg *component.AIConfig, playerEnt ecs.Entity, playerFound bool, playerX, playerY float64) *AIActionContext {
	animComp, _ := ecs.Get(w, ent, component.AnimationComponent.Kind())
	spriteComp, _ := ecs.Get(w, ent, component.SpriteComponent.Kind())
	bodyComp, _ := ecs.Get(w, ent, component.PhysicsBodyComponent.Kind())

	getPos := func() (x, y float64) {
		return phaseEntityPosition(w, ent)
	}

	return &AIActionContext{
		World:        w,
		Entity:       ent,
		AI:           ai,
		State:        state,
		Context:      aiCtx,
		Config:       cfg,
		PlayerFound:  playerFound,
		PlayerX:      playerX,
		PlayerY:      playerY,
		PlayerEntity: playerEnt,
		GetPosition:  getPos,
		GetVelocity: func() (x, y float64) {
			if bodyComp == nil || bodyComp.Body == nil {
				return 0, 0
			}
			vel := bodyComp.Body.Velocity()
			return vel.X, vel.Y
		},
		SetVelocity: func(x, y float64) {
			if bodyComp == nil || bodyComp.Body == nil {
				return
			}
			bodyComp.Body.SetVelocityVector(cp.Vector{X: x, Y: y})
		},
		ChangeAnimation: func(animation string) {
			if animComp == nil || spriteComp == nil {
				return
			}
			def, ok := animComp.Defs[animation]
			if !ok || animComp.Sheet == nil {
				return
			}
			animComp.Current = animation
			animComp.Frame = 0
			animComp.FrameTimer = 0
			animComp.Playing = true
			rect := image.Rect(def.ColStart*def.FrameW, def.Row*def.FrameH, def.ColStart*def.FrameW+def.FrameW, def.Row*def.FrameH+def.FrameH)
			spriteComp.Image = animComp.Sheet.SubImage(rect).(*ebiten.Image)
		},
		FacingLeft: func(facingLeft bool) {
			if spriteComp == nil {
				return
			}
			spriteComp.FacingLeft = facingLeft
		},
		EnqueueEvent: func(ev component.EventID) {
			if ev == "" {
				return
			}
			q, ok := ecs.Get(w, ent, component.AIEventQueueComponent.Kind())
			if !ok || q == nil {
				q = &component.AIEventQueue{Events: make([]string, 0, 2)}
			}
			q.Events = append(q.Events, string(ev))
			_ = ecs.Add(w, ent, component.AIEventQueueComponent.Kind(), q)
		},
	}
}

func phaseEntityPosition(w *ecs.World, ent ecs.Entity) (float64, float64) {
	if pb, ok := ecs.Get(w, ent, component.PhysicsBodyComponent.Kind()); ok && pb != nil && pb.Body != nil {
		p := pb.Body.Position()
		return p.X, p.Y
	}
	if tf, ok := ecs.Get(w, ent, component.TransformComponent.Kind()); ok && tf != nil {
		return tf.X, tf.Y
	}
	return 0, 0
}

func copyTransitions(src map[string][]map[string]any) map[string][]map[string]any {
	out := make(map[string][]map[string]any, len(src))
	for from, entries := range src {
		copied := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			dup := make(map[string]any, len(entry))
			for k, v := range entry {
				dup[k] = v
			}
			copied = append(copied, dup)
		}
		out[from] = copied
	}
	return out
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	default:
		return 0, false
	}
}

func toBool(v any) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}
