package system

import (
	"fmt"
	"strings"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type BossSystem struct{}

func NewBossSystem() *BossSystem { return &BossSystem{} }

func (s *BossSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach3(w,
		component.BossComponent.Kind(),
		component.BossRuntimeComponent.Kind(),
		component.HealthComponent.Kind(),
		func(e ecs.Entity, boss *component.Boss, runtime *component.BossRuntime, hp *component.Health) {
			if boss == nil || runtime == nil || hp == nil || len(boss.Phases) == 0 {
				return
			}

			if !runtime.Initialized {
				runtime.Initialized = true
				runtime.CurrentPhase = 0
				runtime.PatternIndex = 0
				runtime.Cooldown = 0
				s.applyPhaseEnter(w, e, &boss.Phases[0])
			}

			if runtime.CurrentPhase < 0 {
				runtime.CurrentPhase = 0
			}
			if runtime.CurrentPhase >= len(boss.Phases) {
				runtime.CurrentPhase = len(boss.Phases) - 1
			}

			// Sequential phase progression by HP threshold. A phase N (N>0)
			// activates when health is <= its hp_trigger.
			for runtime.CurrentPhase+1 < len(boss.Phases) {
				next := &boss.Phases[runtime.CurrentPhase+1]
				if hp.Current > next.HPTrigger {
					break
				}
				runtime.CurrentPhase++
				runtime.PatternIndex = 0
				runtime.Cooldown = 0
				s.applyPhaseEnter(w, e, next)
			}

			phase := &boss.Phases[runtime.CurrentPhase]
			if runtime.Cooldown > 0 {
				runtime.Cooldown--
				return
			}

			if len(phase.Patterns) == 0 {
				return
			}

			idx := s.selectPatternIndex(e, phase.PatternMode, runtime.PatternIndex, len(phase.Patterns))
			if idx < 0 || idx >= len(phase.Patterns) {
				idx = 0
			}
			pattern := phase.Patterns[idx]
			runtime.PatternIndex++

			s.applyActions(w, e, pattern.Actions)

			cooldown := pattern.CooldownFrames
			if cooldown <= 0 {
				cooldown = 90
			}
			runtime.Cooldown = cooldown
		})
}

func (s *BossSystem) applyPhaseEnter(w *ecs.World, e ecs.Entity, phase *component.BossPhase) {
	if phase == nil {
		return
	}
	s.applyActions(w, e, phase.OnEnter)
	s.applyActions(w, e, phase.Arena)
}

func (s *BossSystem) selectPatternIndex(e ecs.Entity, mode string, cursor int, n int) int {
	if n <= 0 {
		return 0
	}
	if strings.EqualFold(mode, "random") {
		seed := cursor*1103515245 + int(uint64(e)%2147483647)
		if seed < 0 {
			seed = -seed
		}
		return seed % n
	}
	if cursor < 0 {
		cursor = 0
	}
	return cursor % n
}

func (s *BossSystem) applyActions(w *ecs.World, e ecs.Entity, actions []map[string]any) {
	for _, entry := range actions {
		for key, arg := range entry {
			s.applyAction(w, e, key, arg)
		}
	}
}

func (s *BossSystem) applyAction(w *ecs.World, e ecs.Entity, key string, arg any) {
	switch key {
	case "emit_ai_event":
		s.queueAIEvent(w, e, asString(arg))
	case "emit_ai_events":
		s.queueAIEvents(w, e, arg)
	case "run_fsm_event":
		s.queueAIEvent(w, e, asString(arg))
	case "run_fsm_events":
		s.queueAIEvents(w, e, arg)
	case "set_ai":
		s.setAIStats(w, e, arg)
	case "arena_set_active":
		s.applyArenaToggle(w, arg, func(n *component.ArenaNode, v bool) { n.Active = v })
	case "arena_set_hazard":
		s.applyArenaToggle(w, arg, func(n *component.ArenaNode, v bool) { n.HazardEnabled = v })
	case "arena_set_transition":
		s.applyArenaToggle(w, arg, func(n *component.ArenaNode, v bool) { n.TransitionEnabled = v })
	case "print":
		fmt.Println("boss:", asString(arg))
	default:
		fmt.Println("boss: unsupported coordinator action", key)
	}
}

func (s *BossSystem) setAIStats(w *ecs.World, e ecs.Entity, arg any) {
	m, ok := arg.(map[string]any)
	if !ok {
		return
	}
	ai, ok := ecs.Get(w, e, component.AIComponent.Kind())
	if !ok || ai == nil {
		return
	}
	if v, ok := asFloatFromMap(m, "move_speed"); ok {
		ai.MoveSpeed = v
	}
	if v, ok := asFloatFromMap(m, "follow_range"); ok {
		ai.FollowRange = v
	}
	if v, ok := asFloatFromMap(m, "attack_range"); ok {
		ai.AttackRange = v
	}
	if v, ok := asIntFromMap(m, "attack_frames"); ok {
		ai.AttackFrames = v
	}
	_ = ecs.Add(w, e, component.AIComponent.Kind(), ai)
}
func (s *BossSystem) queueAIEvent(w *ecs.World, e ecs.Entity, event string) {
	if w == nil || event == "" {
		return
	}
	q, ok := ecs.Get(w, e, component.AIEventQueueComponent.Kind())
	if !ok || q == nil {
		q = &component.AIEventQueue{Events: make([]string, 0, 2)}
	}
	q.Events = append(q.Events, event)
	_ = ecs.Add(w, e, component.AIEventQueueComponent.Kind(), q)
}

func (s *BossSystem) queueAIEvents(w *ecs.World, e ecs.Entity, arg any) {
	switch v := arg.(type) {
	case []any:
		for _, it := range v {
			s.queueAIEvent(w, e, asString(it))
		}
	case []string:
		for _, it := range v {
			s.queueAIEvent(w, e, it)
		}
	default:
		s.queueAIEvent(w, e, asString(arg))
	}
}

func (s *BossSystem) applyArenaToggle(w *ecs.World, arg any, apply func(n *component.ArenaNode, v bool)) {
	m, ok := arg.(map[string]any)
	if !ok || apply == nil {
		return
	}
	group := asString(m["group"])
	if group == "" {
		return
	}
	value, ok := asBoolValue(m["value"])
	if !ok {
		if v, ok2 := asBoolValue(m["active"]); ok2 {
			value = v
			ok = true
		} else if v, ok2 := asBoolValue(m["enabled"]); ok2 {
			value = v
			ok = true
		}
	}
	if !ok {
		return
	}

	ecs.ForEach(w, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
		if node == nil || node.Group != group {
			return
		}
		apply(node, value)
		_ = ecs.Add(w, ent, component.ArenaNodeComponent.Kind(), node)
	})
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func asBoolValue(v any) (bool, bool) {
	switch b := v.(type) {
	case bool:
		return b, true
	case string:
		if strings.EqualFold(b, "true") {
			return true, true
		}
		if strings.EqualFold(b, "false") {
			return false, true
		}
	}
	return false, false
}

func asFloatValue(v any) (float64, bool) {
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
	}
	return 0, false
}

func asIntValue(v any) (int, bool) {
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
	}
	return 0, false
}

func asFloatFromMap(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	return asFloatValue(v)
}

func asIntFromMap(m map[string]any, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	return asIntValue(v)
}
