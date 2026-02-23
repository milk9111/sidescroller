package system

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

type aiScriptRuntime struct {
	scriptPath  string
	compiled    *tengo.Compiled
	stateData   *tengo.Map
	initial     component.StateID
	initialized bool
	pending     component.StateID
}

const aiLifecycleDispatchScript = `
if __phase == "enter" {
	onEnter(__engine, __state, __current_state)
} else if __phase == "update" {
	update(__engine, __state, __current_state)
} else if __phase == "exit" {
	onExit(__engine, __state, __current_state)
}
`

func (e *AISystem) updateFromScript(ctx *AIActionContext, spec *component.AIFSMSpec, events []component.EventID) {
	if e == nil || ctx == nil || ctx.State == nil || spec == nil {
		return
	}

	rt, err := e.getScriptRuntime(ctx.Entity, spec)
	if err != nil {
		fmt.Printf("ai: entity=%d load scripted FSM error: %v\n", ctx.Entity, err)
		return
	}

	if ctx.State.Current == "" {
		ctx.State.Current = rt.initial
	}

	eventSet := make(map[string]bool, len(events))
	for _, ev := range events {
		if ev == "" {
			continue
		}
		eventSet[string(ev)] = true
	}

	engine := buildAIScriptEngine(ctx, rt, eventSet)
	if !rt.initialized {
		if err := rt.runPhase("enter", ctx.State.Current, engine); err != nil {
			fmt.Printf("ai: entity=%d script onEnter error: %v\n", ctx.Entity, err)
			return
		}
		rt.initialized = true
	}

	if err := rt.runPhase("update", ctx.State.Current, engine); err != nil {
		fmt.Printf("ai: entity=%d script update error: %v\n", ctx.Entity, err)
		return
	}

	if rt.pending == "" || rt.pending == ctx.State.Current {
		rt.pending = ""
		return
	}

	prev := ctx.State.Current
	if err := rt.runPhase("exit", prev, engine); err != nil {
		fmt.Printf("ai: entity=%d script onExit error: %v\n", ctx.Entity, err)
		return
	}

	ctx.State.Current = rt.pending
	rt.pending = ""

	if err := rt.runPhase("enter", ctx.State.Current, engine); err != nil {
		fmt.Printf("ai: entity=%d script onEnter error: %v\n", ctx.Entity, err)
	}
}

func (e *AISystem) getScriptRuntime(ent ecs.Entity, spec *component.AIFSMSpec) (*aiScriptRuntime, error) {
	if e == nil || spec == nil || !spec.ScriptLifecycle || strings.TrimSpace(spec.ScriptPath) == "" {
		return nil, fmt.Errorf("invalid scripted FSM spec")
	}
	if e.scriptCache == nil {
		e.scriptCache = map[ecs.Entity]*aiScriptRuntime{}
	}

	if rt, ok := e.scriptCache[ent]; ok && rt != nil && rt.scriptPath == spec.ScriptPath {
		return rt, nil
	}

	scriptBytes, err := prefabs.LoadScript(spec.ScriptPath)
	if err != nil {
		return nil, err
	}

	src := string(scriptBytes) + "\n" + aiLifecycleDispatchScript
	script := tengo.NewScript([]byte(src))
	_ = script.Add("__phase", "")
	_ = script.Add("__engine", map[string]any{})
	_ = script.Add("__state", map[string]any{})
	_ = script.Add("__current_state", "")

	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	compiled, err := script.Compile()
	if err != nil {
		return nil, err
	}

	rt := &aiScriptRuntime{
		scriptPath: spec.ScriptPath,
		compiled:   compiled,
		stateData:  &tengo.Map{Value: map[string]tengo.Object{}},
		initial:    component.StateID("idle"),
	}

	// Resolve optional initial state from script global `initial_state`.
	noop := &tengo.ImmutableMap{Value: map[string]tengo.Object{}}
	if err := rt.runPhase("noop", rt.initial, noop); err != nil {
		return nil, err
	}
	if compiled.IsDefined("initial_state") {
		s := strings.TrimSpace(compiled.Get("initial_state").String())
		if s != "" {
			rt.initial = component.StateID(s)
		}
	}

	e.scriptCache[ent] = rt
	return rt, nil
}

func (rt *aiScriptRuntime) runPhase(phase string, current component.StateID, engine *tengo.ImmutableMap) error {
	if rt == nil || rt.compiled == nil {
		return fmt.Errorf("nil script runtime")
	}
	if engine == nil {
		engine = &tengo.ImmutableMap{Value: map[string]tengo.Object{}}
	}
	if err := rt.compiled.Set("__phase", phase); err != nil {
		return err
	}
	if err := rt.compiled.Set("__engine", engine); err != nil {
		return err
	}
	if err := rt.compiled.Set("__state", rt.stateData); err != nil {
		return err
	}
	if err := rt.compiled.Set("__current_state", string(current)); err != nil {
		return err
	}
	return rt.compiled.Run()
}

func buildAIScriptEngine(ctx *AIActionContext, rt *aiScriptRuntime, eventSet map[string]bool) *tengo.ImmutableMap {
	values := map[string]tengo.Object{}

	values["transition"] = &tengo.UserFunction{Name: "transition", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if rt == nil || len(args) < 1 {
			return tengo.FalseValue, nil
		}
		name := strings.TrimSpace(objectAsString(args[0]))
		if name == "" {
			return tengo.FalseValue, nil
		}
		rt.pending = component.StateID(name)
		return tengo.TrueValue, nil
	}}

	values["emit"] = &tengo.UserFunction{Name: "emit", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if ctx == nil || ctx.EnqueueEvent == nil || len(args) < 1 {
			return tengo.FalseValue, nil
		}
		name := strings.TrimSpace(objectAsString(args[0]))
		if name == "" {
			return tengo.FalseValue, nil
		}
		ctx.EnqueueEvent(component.EventID(name))
		return tengo.TrueValue, nil
	}}

	values["event"] = &tengo.UserFunction{Name: "event", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.FalseValue, nil
		}
		name := strings.TrimSpace(objectAsString(args[0]))
		if name == "" {
			return tengo.FalseValue, nil
		}
		if eventSet[name] {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}}

	values["consume_event"] = &tengo.UserFunction{Name: "consume_event", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.FalseValue, nil
		}
		name := strings.TrimSpace(objectAsString(args[0]))
		if name == "" {
			return tengo.FalseValue, nil
		}
		if eventSet[name] {
			delete(eventSet, name)
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}}

	values["get_player_position"] = &tengo.UserFunction{Name: "get_player_position", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if ctx == nil {
			return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, nil
		}

		return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: ctx.PlayerX}, &tengo.Float{Value: ctx.PlayerY}}}, nil
	}}

	values["get_position"] = &tengo.UserFunction{Name: "get_position", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if ctx == nil || ctx.GetPosition == nil {
			return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, nil
		}

		x, y := ctx.GetPosition()
		return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: x}, &tengo.Float{Value: y}}}, nil
	}}

	values["get_facing_left"] = &tengo.UserFunction{Name: "get_facing_left", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if ctx == nil {
			return tengo.FalseValue, nil
		}

		spriteComp, ok := ecs.Get(ctx.World, ctx.Entity, component.SpriteComponent.Kind())
		if !ok {
			return tengo.FalseValue, nil
		}

		if spriteComp.FacingLeft {
			return tengo.TrueValue, nil
		} else {
			return tengo.FalseValue, nil
		}
	}}

	values["action"] = &tengo.UserFunction{Name: "action", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if ctx == nil || len(args) < 1 {
			return tengo.FalseValue, nil
		}
		name := strings.TrimSpace(objectAsString(args[0]))
		if name == "" {
			return tengo.FalseValue, nil
		}
		maker, ok := actionRegistry[name]
		if !ok {
			return tengo.FalseValue, nil
		}
		var arg any
		if len(args) > 1 {
			arg = objectToAny(args[1])
		}
		maker(arg)(ctx)
		return tengo.TrueValue, nil
	}}

	for name, maker := range actionRegistry {
		actionName := name
		makeAction := maker
		values[actionName] = &tengo.UserFunction{Name: actionName, Value: func(args ...tengo.Object) (tengo.Object, error) {
			if ctx == nil {
				return tengo.FalseValue, nil
			}
			var arg any
			if len(args) > 0 {
				arg = objectToAny(args[0])
			}
			makeAction(arg)(ctx)
			return tengo.TrueValue, nil
		}}
	}

	for name, maker := range transitionRegistry {
		transitionName := name
		makeTransition := maker
		values[transitionName] = &tengo.UserFunction{Name: transitionName, Value: func(args ...tengo.Object) (tengo.Object, error) {
			if ctx == nil {
				return tengo.FalseValue, nil
			}

			var arg any
			if len(args) > 0 {
				arg = objectToAny(args[0])
			}

			v := makeTransition(arg)(ctx)

			if v {
				return tengo.TrueValue, nil
			} else {
				return tengo.FalseValue, nil
			}
		}}
	}

	return &tengo.ImmutableMap{Value: values}
}

func objectAsString(obj tengo.Object) string {
	if obj == nil {
		return ""
	}
	switch v := obj.(type) {
	case *tengo.String:
		return v.Value
	default:
		return strings.Trim(v.String(), "\"")
	}
}

func objectToAny(obj tengo.Object) any {
	if obj == nil {
		return nil
	}

	switch v := obj.(type) {
	case *tengo.String:
		return v.Value
	case *tengo.Int:
		return int(v.Value)
	case *tengo.Float:
		return v.Value
	case *tengo.Bool:
		return !v.IsFalsy()
	case *tengo.Array:
		out := make([]any, 0, len(v.Value))
		for _, item := range v.Value {
			out = append(out, objectToAny(item))
		}
		return out
	case *tengo.Map:
		out := make(map[string]any, len(v.Value))
		for k, item := range v.Value {
			out[k] = objectToAny(item)
		}
		return out
	case *tengo.ImmutableMap:
		out := make(map[string]any, len(v.Value))
		for k, item := range v.Value {
			out[k] = objectToAny(item)
		}
		return out
	case *tengo.Undefined:
		return nil
	default:
		return v.String()
	}
}
