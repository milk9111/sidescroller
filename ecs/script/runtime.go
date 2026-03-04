package script

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	scriptmodule "github.com/milk9111/sidescroller/ecs/script/module"
	"github.com/milk9111/sidescroller/prefabs"
)

var scriptImportPattern = regexp.MustCompile(`(?m)import\s*\(\s*"([^"]+)"\s*\)`)
var scriptImportAssignPattern = regexp.MustCompile(`(?m)^[ \t]*([A-Za-z_][A-Za-z0-9_]*)[ \t]*:=[ \t]*import\([ \t]*"([^"]+)"[ \t]*\)[ \t]*$`)
var scriptImportCallLinePattern = regexp.MustCompile(`(?m)^[ \t]*import\([ \t]*"([^"]+)"[ \t]*\)[ \t]*$`)

type Runtime struct {
	runtimes       map[ecs.Entity]*entityRuntime
	modules        map[string]scriptmodule.Module
	world          *ecs.World
	byGameEntityID map[string]ecs.Entity
}

type entityRuntime struct {
	scriptPath    string
	compiled      *tengo.Compiled
	state         *tengo.Map
	hasOnStart    bool
	hasOnUpdate   bool
	subscriptions map[string][]subscription
}

type subscription struct {
	sourceGameEntity string
	callback         tengo.Object
}

const lifecycleStartDispatch = `
if __phase == "start" {
	on_start(__engine, __state)
}
`

const lifecycleUpdateDispatch = `
if __phase == "update" {
	on_update(__engine, __state)
}
`

const lifecycleSignalDispatch = `
if __phase == "signal" {
	__signal_callback(__signal_name, __signal_source_game_entity)
}
`

func NewRuntime() *Runtime {
	r := &Runtime{
		runtimes:       map[ecs.Entity]*entityRuntime{},
		modules:        map[string]scriptmodule.Module{},
		byGameEntityID: map[string]ecs.Entity{},
	}
	r.RegisterBuiltinModules()
	return r
}

func (r *Runtime) RegisterBuiltinModules() {
	if r == nil {
		return
	}

	for _, module := range scriptmodule.Builtins() {
		r.RegisterModule(module)
	}
}

func (r *Runtime) RegisterModule(module scriptmodule.Module) {
	if r == nil {
		return
	}

	name := strings.TrimSpace(module.Name)
	if name == "" || module.Build == nil {
		return
	}
	if r.modules == nil {
		r.modules = map[string]scriptmodule.Module{}
	}
	module.Name = name
	r.modules[name] = module
}

func (r *Runtime) Update(w *ecs.World) {
	if r == nil || w == nil {
		return
	}

	for ent := range r.runtimes {
		if !ecs.IsAlive(w, ent) {
			delete(r.runtimes, ent)
		}
	}

	r.rebuildEntityIndex(w)

	ecs.ForEach(w, component.ScriptComponent.Kind(), func(ent ecs.Entity, scriptComp *component.Script) {
		if scriptComp == nil || (strings.TrimSpace(scriptComp.Path) == "" && len(scriptComp.Paths) == 0) {
			return
		}

		rt, err := r.getRuntime(ent, scriptComp)
		if err != nil {
			fmt.Printf("script: entity=%d load runtime error: %v\n", ent, err)
			return
		}

		runtimeComp, ok := ecs.Get(w, ent, component.ScriptRuntimeComponent.Kind())
		if !ok || runtimeComp == nil {
			runtimeComp = &component.ScriptRuntime{}
		}

		if !runtimeComp.Started {
			if err := rt.runPhase("start", r.buildEngineContext(ent)); err != nil {
				fmt.Printf("script: entity=%d on_start error: %v\n", ent, err)
				return
			}
			runtimeComp.Started = true
			_ = ecs.Add(w, ent, component.ScriptRuntimeComponent.Kind(), runtimeComp)
		}

		events := drainScriptSignalQueue(w, ent)
		for _, event := range events {
			rt.dispatchSignal(event)
		}

		if err := rt.runPhase("update", r.buildEngineContext(ent)); err != nil {
			fmt.Printf("script: entity=%d on_update error: %v\n", ent, err)
		}
	})
}

func (r *Runtime) getRuntime(ent ecs.Entity, scriptComp *component.Script) (*entityRuntime, error) {
	if r == nil || scriptComp == nil {
		return nil, fmt.Errorf("invalid script runtime request")
	}
	if r.runtimes == nil {
		r.runtimes = map[ecs.Entity]*entityRuntime{}
	}

	// Determine script paths (support both legacy single Path and new Paths list)
	var paths []string
	if len(scriptComp.Paths) > 0 {
		paths = append([]string(nil), scriptComp.Paths...)
	} else if strings.TrimSpace(scriptComp.Path) != "" {
		paths = []string{strings.TrimSpace(scriptComp.Path)}
	}

	joinedPaths := strings.Join(paths, ";")
	if rt, ok := r.runtimes[ent]; ok && rt != nil && rt.scriptPath == joinedPaths {
		return rt, nil
	}

	rt := &entityRuntime{
		scriptPath:    joinedPaths,
		state:         &tengo.Map{Value: map[string]tengo.Object{}},
		subscriptions: map[string][]subscription{},
	}

	// Load and concatenate script file contents in order.
	// While doing so, extract any import(...) declarations, remove them
	// from each source file, deduplicate the imports, and place a single
	// import block at the top of the final script.
	var sb strings.Builder
	var rawSB strings.Builder
	imports := map[string]string{}
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		b, err := prefabs.LoadScript(p)
		if err != nil {
			return nil, err
		}
		src := string(b)
		rawSB.WriteString(src)
		rawSB.WriteString("\n")

		// find assignment-style import declarations in this file
		assignMatches := scriptImportAssignPattern.FindAllStringSubmatch(src, -1)
		for _, m := range assignMatches {
			if len(m) >= 3 {
				varName := strings.TrimSpace(m[1])
				modName := strings.TrimSpace(m[2])
				if modName == "" {
					continue
				}
				if _, ok := imports[modName]; !ok {
					// prefer the first variable name encountered for this module
					if varName == "" {
						imports[modName] = modName
					} else {
						imports[modName] = varName
					}
				}
			}
		}

		// find bare import(...) lines
		callMatches := scriptImportCallLinePattern.FindAllStringSubmatch(src, -1)
		for _, m := range callMatches {
			if len(m) >= 2 {
				modName := strings.TrimSpace(m[1])
				if modName == "" {
					continue
				}
				if _, ok := imports[modName]; !ok {
					imports[modName] = modName
				}
			}
		}

		// remove import declarations (both assignment lines and bare calls) from this file before appending
		cleaned := scriptImportAssignPattern.ReplaceAllString(src, "")
		cleaned = scriptImportCallLinePattern.ReplaceAllString(cleaned, "")
		sb.WriteString(cleaned)
		sb.WriteString("\n")
	}

	// build deduplicated import block (if any)
	importNames := make([]string, 0, len(imports))
	for name := range imports {
		importNames = append(importNames, name)
	}
	sort.Strings(importNames)

	importBlock := ""
	if len(importNames) > 0 {
		var ib strings.Builder
		for _, mod := range importNames {
			varName := imports[mod]
			if varName == "" {
				varName = mod
			}
			ib.WriteString(varName)
			ib.WriteString(" := import(\"")
			ib.WriteString(mod)
			ib.WriteString("\")\n")
		}
		ib.WriteString("\n")
		importBlock = ib.String()
	}

	// final script string has a single import block (assignment-style) followed by cleaned sources
	scriptString := importBlock + sb.String()
	rawScriptString := rawSB.String()

	startDispatch := ""
	if strings.Contains(scriptString, "on_start") {
		startDispatch = "\n" + lifecycleStartDispatch
	}

	updateDispatch := ""
	if strings.Contains(scriptString, "on_update") {
		updateDispatch = "\n" + lifecycleUpdateDispatch
	}

	src := scriptString + startDispatch + updateDispatch + "\n" + lifecycleSignalDispatch

	script := tengo.NewScript([]byte(src))
	_ = script.Add("__phase", "")
	_ = script.Add("__engine", map[string]any{})
	_ = script.Add("__state", map[string]any{})
	_ = script.Add("__signal_callback", tengo.UndefinedValue)
	_ = script.Add("__signal_name", "")
	_ = script.Add("__signal_source_game_entity", "")
	// Use the raw script content when resolving requested modules so
	// we detect any import(...) usages even though we've removed them
	// from the per-file sources and centralized them above.
	requestedModules := r.resolveRequestedModules(rawScriptString, scriptComp.Modules)
	script.SetImports(r.buildModuleMap(ent, rt, requestedModules))

	compiled, err := script.Compile()
	if err != nil {
		return nil, err
	}

	rt.compiled = compiled
	rt.hasOnStart = startDispatch != ""
	rt.hasOnUpdate = updateDispatch != ""

	r.runtimes[ent] = rt
	return rt, nil
}

func (r *Runtime) resolveRequestedModules(scriptString string, configured []string) []string {
	if len(configured) > 0 {
		return append([]string(nil), configured...)
	}

	if len(r.modules) == 0 || scriptString == "" {
		return nil
	}

	matches := scriptImportPattern.FindAllStringSubmatch(scriptString, -1)
	if len(matches) == 0 {
		return nil
	}

	derived := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if _, ok := r.modules[name]; !ok {
			continue
		}
		derived[name] = true
	}

	if len(derived) == 0 {
		return nil
	}

	names := make([]string, 0, len(derived))
	for name := range derived {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func (r *Runtime) buildModuleMap(owner ecs.Entity, rt *entityRuntime, modules []string) *tengo.ModuleMap {
	moduleMap := stdlib.GetModuleMap(stdlib.AllModuleNames()...).Copy()
	allowed := map[string]bool{}
	if len(modules) > 0 {
		for _, name := range modules {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			allowed[name] = true
		}
	}

	moduleMap.AddBuiltinModule("signals", r.buildSignalsModule(owner, rt))

	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		if len(allowed) > 0 && !allowed[name] {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		plugin := r.modules[name]
		moduleMap.AddBuiltinModule(name, r.buildPluginModule(plugin, owner, owner))
	}

	return moduleMap
}

func (r *Runtime) buildEngineContext(owner ecs.Entity) *tengo.ImmutableMap {
	values := map[string]tengo.Object{}
	values["game_entity_id"] = &tengo.String{Value: r.gameEntityID(owner)}
	values["emit"] = &tengo.UserFunction{Name: "emit", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.FalseValue, nil
		}
		signal := strings.TrimSpace(objectAsString(args[0]))
		if signal == "" {
			return tengo.FalseValue, nil
		}
		target := owner
		if len(args) > 1 {
			targetID := strings.TrimSpace(objectAsString(args[1]))
			resolved, ok := r.resolveEntity(owner, targetID)
			if !ok {
				return tengo.FalseValue, nil
			}
			target = resolved
		}
		if EmitEntitySignal(r.world, target, owner, signal) {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}}
	return &tengo.ImmutableMap{Value: values}
}

func (r *Runtime) buildPluginModule(plugin scriptmodule.Module, owner ecs.Entity, target ecs.Entity) map[string]tengo.Object {
	values := plugin.Build(r.world, r.byGameEntityID, owner, target)
	if values == nil {
		values = map[string]tengo.Object{}
	}

	values["for_entity"] = &tengo.UserFunction{Name: "for_entity", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return &tengo.ImmutableMap{Value: map[string]tengo.Object{}}, nil
		}

		targetID := strings.TrimSpace(objectAsString(args[0]))
		resolved, ok := r.resolveEntity(owner, targetID)
		if !ok {
			return &tengo.ImmutableMap{Value: map[string]tengo.Object{}}, fmt.Errorf("could not find entity %s", targetID)
		}

		return &tengo.ImmutableMap{Value: r.buildPluginModule(plugin, owner, resolved)}, nil
	}}

	return values
}

func (r *Runtime) buildSignalsModule(owner ecs.Entity, rt *entityRuntime) map[string]tengo.Object {
	ownerGameEntityID := r.gameEntityID(owner)
	values := map[string]tengo.Object{}
	values["on"] = &tengo.UserFunction{Name: "on", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if rt == nil || len(args) < 2 {
			return tengo.FalseValue, nil
		}
		signal := strings.TrimSpace(objectAsString(args[0]))
		if signal == "" {
			return tengo.FalseValue, nil
		}
		callback := args[1]
		if !isCallableObject(callback) {
			return tengo.FalseValue, nil
		}
		sourceGameEntity := ownerGameEntityID
		if len(args) > 2 {
			sourceGameEntity = strings.TrimSpace(objectAsString(args[2]))
			if sourceGameEntity == "*" {
				sourceGameEntity = ""
			}
		}
		rt.subscriptions[signal] = append(rt.subscriptions[signal], subscription{sourceGameEntity: sourceGameEntity, callback: callback})
		return tengo.TrueValue, nil
	}}
	values["off"] = &tengo.UserFunction{Name: "off", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if rt == nil || len(args) < 1 {
			return tengo.FalseValue, nil
		}
		signal := strings.TrimSpace(objectAsString(args[0]))
		if signal == "" {
			return tengo.FalseValue, nil
		}
		delete(rt.subscriptions, signal)
		return tengo.TrueValue, nil
	}}
	values["emit"] = &tengo.UserFunction{Name: "emit", Value: func(args ...tengo.Object) (tengo.Object, error) {
		if len(args) < 1 {
			return tengo.FalseValue, nil
		}
		signal := strings.TrimSpace(objectAsString(args[0]))
		if signal == "" {
			return tengo.FalseValue, nil
		}
		target := owner
		if len(args) > 1 {
			targetID := strings.TrimSpace(objectAsString(args[1]))
			resolved, ok := r.resolveEntity(owner, targetID)
			if !ok {
				return tengo.FalseValue, nil
			}
			target = resolved
		}
		if EmitEntitySignal(r.world, target, owner, signal) {
			return tengo.TrueValue, nil
		}
		return tengo.FalseValue, nil
	}}
	return values
}

func (r *Runtime) rebuildEntityIndex(w *ecs.World) {
	r.world = w
	if r.byGameEntityID == nil {
		r.byGameEntityID = map[string]ecs.Entity{}
	}
	for key := range r.byGameEntityID {
		delete(r.byGameEntityID, key)
	}
	if w == nil {
		return
	}
	ecs.ForEach(w, component.GameEntityIDComponent.Kind(), func(ent ecs.Entity, id *component.GameEntityID) {
		if id == nil {
			return
		}
		value := strings.TrimSpace(id.Value)
		if value == "" {
			return
		}
		if _, exists := r.byGameEntityID[value]; exists {
			return
		}
		r.byGameEntityID[value] = ent
	})
}

func (r *Runtime) resolveEntity(owner ecs.Entity, selector string) (ecs.Entity, bool) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return owner, true
	}
	ent, ok := r.byGameEntityID[selector]
	return ent, ok
}

func (r *Runtime) gameEntityID(ent ecs.Entity) string {
	if r == nil || r.world == nil {
		return ""
	}
	id, ok := ecs.Get(r.world, ent, component.GameEntityIDComponent.Kind())
	if !ok || id == nil {
		return ""
	}
	return strings.TrimSpace(id.Value)
}

func (rt *entityRuntime) runPhase(phase string, engineCtx *tengo.ImmutableMap) error {
	if rt == nil || rt.compiled == nil {
		return fmt.Errorf("nil script runtime")
	}
	if phase == "start" && !rt.hasOnStart {
		return nil
	}
	if phase == "update" && !rt.hasOnUpdate {
		return nil
	}
	if engineCtx == nil {
		engineCtx = &tengo.ImmutableMap{Value: map[string]tengo.Object{}}
	}
	if err := rt.compiled.Set("__phase", phase); err != nil {
		return err
	}
	if err := rt.compiled.Set("__engine", engineCtx); err != nil {
		return err
	}
	if err := rt.compiled.Set("__state", rt.state); err != nil {
		return err
	}
	return rt.compiled.Run()
}

func (rt *entityRuntime) dispatchSignal(event component.ScriptSignalEvent) {
	if rt == nil || len(rt.subscriptions) == 0 || strings.TrimSpace(event.Name) == "" {
		return
	}
	subs := rt.subscriptions[event.Name]
	if len(subs) == 0 {
		return
	}
	for _, sub := range subs {
		if sub.sourceGameEntity != "" && sub.sourceGameEntity != strings.TrimSpace(event.SourceGameEntity) {
			continue
		}

		if err := rt.compiled.Set("__phase", "signal"); err != nil {
			fmt.Printf("script: signal set phase error (%s): %v\n", event.Name, err)
			continue
		}
		if err := rt.compiled.Set("__signal_callback", sub.callback); err != nil {
			fmt.Printf("script: signal set callback error (%s): %v\n", event.Name, err)
			continue
		}
		if err := rt.compiled.Set("__signal_name", event.Name); err != nil {
			fmt.Printf("script: signal set name error (%s): %v\n", event.Name, err)
			continue
		}
		if err := rt.compiled.Set("__signal_source_game_entity", event.SourceGameEntity); err != nil {
			fmt.Printf("script: signal set source error (%s): %v\n", event.Name, err)
			continue
		}

		if err := rt.compiled.Run(); err != nil {
			fmt.Printf("script: signal handler error (%s): %v\n", event.Name, err)
		}

		_ = rt.compiled.Set("__signal_callback", tengo.UndefinedValue)
	}
}

func isCallableObject(obj tengo.Object) bool {
	switch obj.(type) {
	case *tengo.CompiledFunction, *tengo.UserFunction, *tengo.BuiltinFunction:
		return true
	default:
		return false
	}
}

func drainScriptSignalQueue(w *ecs.World, ent ecs.Entity) []component.ScriptSignalEvent {
	queue, ok := ecs.Get(w, ent, component.ScriptSignalQueueComponent.Kind())
	if !ok || queue == nil || len(queue.Events) == 0 {
		return nil
	}
	events := append([]component.ScriptSignalEvent(nil), queue.Events...)
	queue.Events = queue.Events[:0]
	_ = ecs.Add(w, ent, component.ScriptSignalQueueComponent.Kind(), queue)
	return events
}

func EmitEntitySignal(w *ecs.World, target ecs.Entity, source ecs.Entity, signalName string) bool {
	if w == nil || !ecs.IsAlive(w, target) {
		return false
	}
	signalName = strings.TrimSpace(signalName)
	if signalName == "" {
		return false
	}

	sourceGameEntity := ""
	if ecs.IsAlive(w, source) {
		if id, ok := ecs.Get(w, source, component.GameEntityIDComponent.Kind()); ok && id != nil {
			sourceGameEntity = strings.TrimSpace(id.Value)
		}
	}

	queue, ok := ecs.Get(w, target, component.ScriptSignalQueueComponent.Kind())
	if !ok || queue == nil {
		queue = &component.ScriptSignalQueue{}
	}
	queue.Events = append(queue.Events, component.ScriptSignalEvent{Name: signalName, SourceGameEntity: sourceGameEntity})
	_ = ecs.Add(w, target, component.ScriptSignalQueueComponent.Kind(), queue)
	return true
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
