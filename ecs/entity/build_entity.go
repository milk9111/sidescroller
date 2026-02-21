package entity

import (
	"errors"
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

type entityPrefabSpec = prefabs.EntityBuildSpec

type buildContext struct {
	PrefabPath string
}

type componentBuildFn func(w *ecs.World, e ecs.Entity, raw any, ctx *buildContext) error

var componentRegistry = map[string]componentBuildFn{
	"player_tag":           addPlayerTag,
	"camera_tag":           addCameraTag,
	"aim_target_tag":       addAimTargetTag,
	"anchor_tag":           addAnchorTag,
	"spike_tag":            addSpikeTag,
	"ai_tag":               addAITag,
	"ai_phase_controller":  addAIPhaseController,
	"ai_phase_runtime":     addAIPhaseRuntime,
	"arena_node":           addArenaNode,
	"player":               addPlayer,
	"input":                addInput,
	"player_state_machine": addPlayerStateMachine,
	"player_collision":     addPlayerCollision,
	"transform":            addTransform,
	"sprite":               addSprite,
	"render_layer":         addRenderLayer,
	"line_render":          addLineRender,
	"camera":               addCamera,
	"ai":                   addAI,
	"pathfinding":          addPathfinding,
	"ai_state":             addAIState,
	"ai_context":           addAIContext,
	"ai_config":            addAIConfig,
	"animation":            addAnimation,
	"audio":                addAudio,
	"collision_layer":      addCollisionLayer,
	"repulsion_layer":      addRepulsionLayer,
	"physics_body":         addPhysicsBody,
	"gravity_scale":        addGravityScale,
	"hazard":               addHazard,
	"health":               addHealth,
	"hitboxes":             addHitboxes,
	"hurtboxes":            addHurtboxes,
	"ai_navigation":        addAINavigation,
	"anchor":               addAnchor,
	"knockbackable":        addKnockbackable,
}

var componentBuildOrder = []string{
	"player_tag",
	"camera_tag",
	"aim_target_tag",
	"anchor_tag",
	"spike_tag",
	"ai_tag",
	"arena_node",
	"player",
	"input",
	"player_state_machine",
	"player_collision",
	"transform",
	"sprite",
	"render_layer",
	"line_render",
	"camera",
	"ai",
	"pathfinding",
	"ai_state",
	"ai_context",
	"ai_config",
	"ai_phase_controller",
	"ai_phase_runtime",
	"animation",
	"audio",
	"collision_layer",
	"repulsion_layer",
	"physics_body",
	"gravity_scale",
	"hazard",
	"health",
	"hitboxes",
	"hurtboxes",
	"ai_navigation",
	"anchor",
}

func BuildEntity(w *ecs.World, prefabPath string) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("build entity: world is nil")
	}

	spec, err := prefabs.LoadEntityBuildSpec(prefabPath)
	if err != nil {
		return 0, fmt.Errorf("build entity: load %q: %w", prefabPath, err)
	}
	if len(spec.Components) == 0 {
		return 0, fmt.Errorf("build entity: prefab %q does not define components", prefabPath)
	}

	e := ecs.CreateEntity(w)
	ctx := &buildContext{PrefabPath: prefabPath}

	remaining := make(map[string]any, len(spec.Components))
	for k, v := range spec.Components {
		remaining[k] = v
	}

	for _, name := range componentBuildOrder {
		raw, ok := remaining[name]
		if !ok {
			continue
		}
		builder, ok := componentRegistry[name]
		if !ok {
			ecs.DestroyEntity(w, e)
			return 0, fmt.Errorf("build entity: %q: no builder for component %q", prefabPath, name)
		}
		if err := builder(w, e, raw, ctx); err != nil {
			ecs.DestroyEntity(w, e)
			return 0, fmt.Errorf("build entity: %q: add %q: %w", prefabPath, name, err)
		}
		delete(remaining, name)
	}

	if len(remaining) > 0 {
		names := make([]string, 0, len(remaining))
		for name := range remaining {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			builder, ok := componentRegistry[name]
			if !ok {
				ecs.DestroyEntity(w, e)
				return 0, fmt.Errorf("build entity: %q: no builder for component %q", prefabPath, name)
			}
			if err := builder(w, e, remaining[name], ctx); err != nil {
				ecs.DestroyEntity(w, e)
				return 0, fmt.Errorf("build entity: %q: add %q: %w", prefabPath, name, err)
			}
		}
	}

	return e, nil
}

func SetEntityTransform(w *ecs.World, e ecs.Entity, x, y, rotation float64) error {
	t, ok := ecs.Get(w, e, component.TransformComponent.Kind())
	if !ok || t == nil {
		t = &component.Transform{ScaleX: 1, ScaleY: 1}
	}
	t.X = x
	t.Y = y
	t.Rotation = rotation
	return ecs.Add(w, e, component.TransformComponent.Kind(), t)
}

func addPlayerTag(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
}

func addCameraTag(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.CameraTagComponent.Kind(), &component.CameraTag{})
}

func addAimTargetTag(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.AimTargetTagComponent.Kind(), &component.AimTargetTag{})
}

func addAnchorTag(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.AnchorTagComponent.Kind(), &component.AnchorTag{})
}

func addSpikeTag(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.SpikeTagComponent.Kind(), &component.SpikeTag{})
}

func addAITag(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.AITagComponent.Kind(), &component.AITag{})
}

type aiPhaseControllerSpec = prefabs.AIPhaseControllerComponentSpec

func addAIPhaseController(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[aiPhaseControllerSpec](raw)
	if err != nil {
		return fmt.Errorf("decode ai_phase_controller spec: %w", err)
	}

	cfg, ok := ecs.Get(w, e, component.AIConfigComponent.Kind())
	if !ok || cfg == nil || cfg.Spec == nil {
		return fmt.Errorf("ai_phase_controller requires ai_config.spec on the same entity")
	}

	baseTransitions := make(map[string][]map[string]any, len(cfg.Spec.Transitions))
	for from, entries := range cfg.Spec.Transitions {
		copied := make([]map[string]any, 0, len(entries))
		for _, entry := range entries {
			dup := make(map[string]any, len(entry))
			for k, v := range entry {
				dup[k] = v
			}
			copied = append(copied, dup)
		}
		baseTransitions[from] = copied
	}

	phases := make([]component.AIPhase, 0, len(spec.Phases))
	for _, phase := range spec.Phases {
		overrides := make(map[string][]map[string]any, len(phase.TransitionOverrides))
		for from, entries := range phase.TransitionOverrides {
			copied := make([]map[string]any, 0, len(entries))
			for _, entry := range entries {
				dup := make(map[string]any, len(entry))
				for k, v := range entry {
					dup[k] = v
				}
				copied = append(copied, dup)
			}
			overrides[from] = copied
		}

		phases = append(phases, component.AIPhase{
			Name:                phase.Name,
			StartWhen:           phase.StartWhen,
			TransitionOverrides: overrides,
			OnEnter:             phase.OnEnter,
		})
	}

	resetState := true
	if spec.ResetStateOnPhaseChange != nil {
		resetState = *spec.ResetStateOnPhaseChange
	}

	if err := ecs.Add(w, e, component.AIPhaseControllerComponent.Kind(), &component.AIPhaseController{
		BaseTransitions:         baseTransitions,
		Phases:                  phases,
		ResetStateOnPhaseChange: resetState,
	}); err != nil {
		return err
	}

	if !ecs.Has(w, e, component.AIPhaseRuntimeComponent.Kind()) {
		if err := ecs.Add(w, e, component.AIPhaseRuntimeComponent.Kind(), &component.AIPhaseRuntime{CurrentPhase: -1}); err != nil {
			return err
		}
	}

	return nil
}

func addAIPhaseRuntime(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	if ecs.Has(w, e, component.AIPhaseRuntimeComponent.Kind()) {
		return nil
	}
	return ecs.Add(w, e, component.AIPhaseRuntimeComponent.Kind(), &component.AIPhaseRuntime{CurrentPhase: -1})
}

type arenaNodeSpec = prefabs.ArenaNodeComponentSpec

func addArenaNode(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[arenaNodeSpec](raw)
	if err != nil {
		return fmt.Errorf("decode arena_node spec: %w", err)
	}

	active := true
	hazardEnabled := true
	transitionEnabled := true
	if spec.Active != nil {
		active = *spec.Active
	}
	if spec.HazardEnabled != nil {
		hazardEnabled = *spec.HazardEnabled
	}
	if spec.TransitionEnabled != nil {
		transitionEnabled = *spec.TransitionEnabled
	}

	if err := ecs.Add(w, e, component.ArenaNodeComponent.Kind(), &component.ArenaNode{
		Group:             spec.Group,
		Active:            active,
		HazardEnabled:     hazardEnabled,
		TransitionEnabled: transitionEnabled,
	}); err != nil {
		return err
	}

	if !ecs.Has(w, e, component.ArenaNodeRuntimeComponent.Kind()) {
		if err := ecs.Add(w, e, component.ArenaNodeRuntimeComponent.Kind(), &component.ArenaNodeRuntime{}); err != nil {
			return err
		}
	}

	return nil
}

type playerSpec = prefabs.PlayerComponentSpec

func addPlayer(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[playerSpec](raw)
	if err != nil {
		return fmt.Errorf("decode player spec: %w", err)
	}
	return ecs.Add(w, e, component.PlayerComponent.Kind(), &component.Player{
		MoveSpeed:            spec.MoveSpeed,
		JumpSpeed:            spec.JumpSpeed,
		JumpHoldFrames:       spec.JumpHoldFrames,
		JumpHoldBoost:        spec.JumpHoldBoost,
		CoyoteFrames:         spec.CoyoteFrames,
		WallGrabFrames:       spec.WallGrabFrames,
		WallSlideSpeed:       spec.WallSlideSpeed,
		WallJumpPush:         spec.WallJumpPush,
		WallJumpFrames:       spec.WallJumpFrames,
		JumpBufferFrames:     spec.JumpBufferFrames,
		AimSlowFactor:        spec.AimSlowFactor,
		HitFreezeFrames:      spec.HitFreezeFrames,
		DamageShakeIntensity: spec.DamageShakeIntensity,
	})
}

func addInput(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.InputComponent.Kind(), &component.Input{})
}

func addPlayerStateMachine(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{})
}

func addPlayerCollision(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.PlayerCollisionComponent.Kind(), &component.PlayerCollision{})
}

type transformSpec = prefabs.TransformComponentSpec

func addTransform(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[transformSpec](raw)
	if err != nil {
		return fmt.Errorf("decode transform spec: %w", err)
	}
	if spec.ScaleX == 0 {
		spec.ScaleX = 1
	}
	if spec.ScaleY == 0 {
		spec.ScaleY = 1
	}
	return ecs.Add(w, e, component.TransformComponent.Kind(), &component.Transform{
		X:        spec.X,
		Y:        spec.Y,
		ScaleX:   spec.ScaleX,
		ScaleY:   spec.ScaleY,
		Rotation: spec.Rotation,
	})
}

type spriteSpec = prefabs.SpriteComponentSpec

func addSprite(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[spriteSpec](raw)
	if err != nil {
		return fmt.Errorf("decode sprite spec: %w", err)
	}

	var sprite component.Sprite
	if spec.Image != "" {
		img, err := assets.LoadImage(spec.Image)
		if err != nil {
			return fmt.Errorf("load image %q: %w", spec.Image, err)
		}
		sprite.Image = img
	}

	sprite.UseSource = spec.UseSource
	sprite.OriginX = spec.OriginX
	sprite.OriginY = spec.OriginY
	if sprite.OriginX == 0 && sprite.OriginY == 0 && spec.CenterOriginIfZero && sprite.Image != nil {
		w, h := sprite.Image.Bounds().Dx(), sprite.Image.Bounds().Dy()
		sprite.OriginX = float64(w) / 2
		sprite.OriginY = float64(h) / 2
	}
	sprite.FacingLeft = spec.FacingLeft

	return ecs.Add(w, e, component.SpriteComponent.Kind(), &sprite)
}

type renderLayerSpec = prefabs.RenderLayerComponentSpec

func addRenderLayer(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[renderLayerSpec](raw)
	if err != nil {
		return fmt.Errorf("decode render layer spec: %w", err)
	}
	return ecs.Add(w, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: spec.Index})
}

type lineRenderSpec = prefabs.LineRenderComponentSpec

func addLineRender(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[lineRenderSpec](raw)
	if err != nil {
		return fmt.Errorf("decode line render spec: %w", err)
	}
	if spec.Width <= 0 {
		spec.Width = 1
	}
	c := color.Color(color.RGBA{R: 255, A: 255})
	if spec.Color != "" {
		parsed, err := parseHexColor(spec.Color)
		if err != nil {
			return fmt.Errorf("parse line render color: %w", err)
		}
		c = parsed
	}
	return ecs.Add(w, e, component.LineRenderComponent.Kind(), &component.LineRender{
		StartX:    spec.StartX,
		StartY:    spec.StartY,
		EndX:      spec.EndX,
		EndY:      spec.EndY,
		Width:     spec.Width,
		Color:     c,
		AntiAlias: spec.AntiAlias,
	})
}

type cameraSpec = prefabs.CameraComponentSpec

func addCamera(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[cameraSpec](raw)
	if err != nil {
		return fmt.Errorf("decode camera spec: %w", err)
	}
	if spec.Smoothness == 0 {
		spec.Smoothness = 0.15
	}
	if spec.LookOffset == 0 {
		spec.LookOffset = 48
	}
	if spec.LookSmooth == 0 {
		spec.LookSmooth = 0.15
	}
	return ecs.Add(w, e, component.CameraComponent.Kind(), &component.Camera{
		TargetName: spec.TargetName,
		Zoom:       spec.Zoom,
		Smoothness: spec.Smoothness,
		LookOffset: spec.LookOffset,
		LookSmooth: spec.LookSmooth,
	})
}

type aiSpec = prefabs.AIComponentSpec

func addAI(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[aiSpec](raw)
	if err != nil {
		return fmt.Errorf("decode AI spec: %w", err)
	}
	return ecs.Add(w, e, component.AIComponent.Kind(), &component.AI{
		MoveSpeed:    spec.MoveSpeed,
		FollowRange:  spec.FollowRange,
		AttackRange:  spec.AttackRange,
		AttackFrames: spec.AttackFrames,
	})
}

type pathfindingSpec = prefabs.PathfindingComponentSpec

func addPathfinding(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[pathfindingSpec](raw)
	if err != nil {
		return fmt.Errorf("decode pathfinding spec: %w", err)
	}
	return ecs.Add(w, e, component.PathfindingComponent.Kind(), &component.Pathfinding{
		GridSize:      spec.GridSize,
		RepathFrames:  spec.RepathFrames,
		DebugNodeSize: spec.DebugNodeSize,
	})
}

func addAIState(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.AIStateComponent.Kind(), &component.AIState{})
}

func addAIContext(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.AIContextComponent.Kind(), &component.AIContext{})
}

type aiFSMStateYAMLSpec = prefabs.AIFSMEmbeddedStateSpec

type aiFSMYAMLSpec = prefabs.AIFSMEmbeddedSpec

type aiConfigSpec = prefabs.AIConfigComponentSpec

var errNoDeclarativeScriptFSM = errors.New("no declarative script fsm")

func addAIConfig(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[aiConfigSpec](raw)
	if err != nil {
		return fmt.Errorf("decode AI config spec: %w", err)
	}

	var outSpec *component.AIFSMSpec
	if spec.Script != "" {
		parsed, err := loadAIFSMSpecFromScript(spec.Script)
		if err != nil {
			if errors.Is(err, errNoDeclarativeScriptFSM) {
				outSpec = &component.AIFSMSpec{ScriptPath: spec.Script, ScriptLifecycle: true}
			} else {
				return fmt.Errorf("load AI FSM script %q: %w", spec.Script, err)
			}
		} else {
			outSpec = parsed
		}
	} else if spec.Spec != nil {
		states := make(map[string]component.AIFSMStateSpec, len(spec.Spec.States))
		for name, st := range spec.Spec.States {
			states[name] = component.AIFSMStateSpec{OnEnter: st.OnEnter, While: st.While, OnExit: st.OnExit}
		}
		outSpec = &component.AIFSMSpec{
			Initial:     spec.Spec.Initial,
			States:      states,
			Transitions: spec.Spec.Transitions,
		}
	}

	return ecs.Add(w, e, component.AIConfigComponent.Kind(), &component.AIConfig{FSM: spec.FSM, Spec: outSpec})
}

func loadAIFSMSpecFromScript(scriptName string) (*component.AIFSMSpec, error) {
	scriptBytes, err := prefabs.LoadScript(scriptName)
	if err != nil {
		return nil, err
	}

	script := tengo.NewScript(scriptBytes)
	script.SetImports(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

	compiled, err := script.Run()
	if err != nil {
		return nil, err
	}

	raw, err := extractScriptFSMRaw(compiled)
	if err != nil {
		return nil, err
	}

	return decodeAIFSMSpec(raw)
}

func extractScriptFSMRaw(compiled *tengo.Compiled) (map[string]any, error) {
	if compiled == nil {
		return nil, fmt.Errorf("script compile returned nil program")
	}

	if fsm := compiled.Get("fsm"); fsm != nil && !fsm.IsUndefined() {
		m, ok := toStringAnyMap(fsm.Value())
		if !ok {
			return nil, fmt.Errorf("script global 'fsm' must be a map")
		}
		return m, nil
	}

	initial := compiled.Get("initial")
	states := compiled.Get("states")
	transitions := compiled.Get("transitions")
	if initial == nil || states == nil || transitions == nil || initial.IsUndefined() || states.IsUndefined() || transitions.IsUndefined() {
		return nil, errNoDeclarativeScriptFSM
	}

	return map[string]any{
		"initial":     initial.Value(),
		"states":      states.Value(),
		"transitions": transitions.Value(),
	}, nil
}

func decodeAIFSMSpec(raw map[string]any) (*component.AIFSMSpec, error) {
	initial, _ := raw["initial"].(string)
	if strings.TrimSpace(initial) == "" {
		return nil, fmt.Errorf("missing 'initial' state")
	}

	statesRaw, ok := toStringAnyMap(raw["states"])
	if !ok {
		return nil, fmt.Errorf("'states' must be a map")
	}

	transitionsRaw, ok := toStringAnyMap(raw["transitions"])
	if !ok {
		return nil, fmt.Errorf("'transitions' must be a map")
	}

	states := make(map[string]component.AIFSMStateSpec, len(statesRaw))
	for name, stateAny := range statesRaw {
		stateMap, ok := toStringAnyMap(stateAny)
		if !ok {
			return nil, fmt.Errorf("state %q must be a map", name)
		}

		onEnter, err := toActionList(stateMap["on_enter"])
		if err != nil {
			return nil, fmt.Errorf("state %q on_enter: %w", name, err)
		}
		whileActions, err := toActionList(stateMap["while"])
		if err != nil {
			return nil, fmt.Errorf("state %q while: %w", name, err)
		}
		onExit, err := toActionList(stateMap["on_exit"])
		if err != nil {
			return nil, fmt.Errorf("state %q on_exit: %w", name, err)
		}

		states[name] = component.AIFSMStateSpec{
			OnEnter: onEnter,
			While:   whileActions,
			OnExit:  onExit,
		}
	}

	transitions := make(map[string][]map[string]any, len(transitionsRaw))
	for from, listAny := range transitionsRaw {
		entries, err := toTransitionList(listAny)
		if err != nil {
			return nil, fmt.Errorf("transitions.%s: %w", from, err)
		}
		transitions[from] = entries
	}

	return &component.AIFSMSpec{
		Initial:     initial,
		States:      states,
		Transitions: transitions,
	}, nil
}

func toActionList(v any) ([]map[string]any, error) {
	if v == nil {
		return nil, nil
	}

	items, ok := toAnySlice(v)
	if !ok {
		return nil, fmt.Errorf("must be an array")
	}

	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := toStringAnyMap(item)
		if !ok {
			return nil, fmt.Errorf("entry %v must be a map", item)
		}
		out = append(out, m)
	}

	return out, nil
}

func toTransitionList(v any) ([]map[string]any, error) {
	if v == nil {
		return nil, nil
	}

	if m, ok := toStringAnyMap(v); ok {
		out := make([]map[string]any, 0, len(m))
		for k, val := range m {
			out = append(out, map[string]any{k: val})
		}
		return out, nil
	}

	items, ok := toAnySlice(v)
	if !ok {
		return nil, fmt.Errorf("must be an array or a map")
	}

	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := toStringAnyMap(item)
		if !ok {
			return nil, fmt.Errorf("entry %v must be a map", item)
		}
		out = append(out, m)
	}

	return out, nil
}

func toStringAnyMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	default:
		return nil, false
	}
}

func toAnySlice(v any) ([]any, bool) {
	switch s := v.(type) {
	case []any:
		return s, true
	default:
		return nil, false
	}
}

type animationDefSpec = prefabs.AnimationDefComponentSpec

type animationSpec = prefabs.AnimationComponentSpec

func addAnimation(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[animationSpec](raw)
	if err != nil {
		return fmt.Errorf("decode animation spec: %w", err)
	}
	sheet, err := assets.LoadImage(spec.Sheet)
	if err != nil {
		return fmt.Errorf("load animation sheet %q: %w", spec.Sheet, err)
	}

	defs := make(map[string]component.AnimationDef, len(spec.Defs))
	for name, def := range spec.Defs {
		defs[name] = component.AnimationDef{
			Row:        def.Row,
			ColStart:   def.ColStart,
			FrameCount: def.FrameCount,
			FrameW:     def.FrameW,
			FrameH:     def.FrameH,
			FPS:        def.FPS,
			Loop:       def.Loop,
		}
	}

	playing := spec.Playing
	if m, ok := raw.(map[string]any); ok {
		if _, has := m["playing"]; !has {
			playing = true
		}
	}

	return ecs.Add(w, e, component.AnimationComponent.Kind(), &component.Animation{
		Sheet:      sheet,
		Defs:       defs,
		Current:    spec.Current,
		Frame:      spec.Frame,
		FrameTimer: spec.FrameTimer,
		Playing:    playing,
	})
}

type audioClipSpec = prefabs.AudioClipSpec

type audioSpec = prefabs.AudioComponentSpec

func addAudio(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[audioSpec](raw)
	if err != nil {
		return fmt.Errorf("decode audio spec: %w", err)
	}
	if len(spec.Clips) == 0 {
		return nil
	}
	comp, err := buildAudioComponentFromSpec(spec.Clips)
	if err != nil {
		return fmt.Errorf("build audio component from spec: %w", err)
	}
	if comp == nil {
		return nil
	}
	for _, name := range spec.Autoplay {
		for i := range comp.Names {
			if comp.Names[i] == name {
				comp.Play[i] = true
			}
		}
	}
	return ecs.Add(w, e, component.AudioComponent.Kind(), comp)
}

type physicsBodySpec = prefabs.PhysicsBodyComponentSpec

func addPhysicsBody(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[physicsBodySpec](raw)
	if err != nil {
		return fmt.Errorf("decode physics body spec: %w", err)
	}
	if spec.DefaultWidth <= 0 {
		spec.DefaultWidth = 32
	}
	if spec.DefaultHeight <= 0 {
		spec.DefaultHeight = 32
	}

	width := spec.Width
	height := spec.Height
	if spec.ScaleWithTransform {
		if tr, ok := ecs.Get(w, e, component.TransformComponent.Kind()); ok && tr != nil {
			width *= tr.ScaleX
			height *= tr.ScaleY
		}
	}
	if width == 0 {
		width = spec.DefaultWidth
	}
	if height == 0 {
		height = spec.DefaultHeight
	}
	if !spec.Static && spec.Mass == 0 {
		spec.Mass = 1
	}

	return ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{
		Width:        width,
		Height:       height,
		Radius:       spec.Radius,
		Mass:         spec.Mass,
		Friction:     spec.Friction,
		Elasticity:   spec.Elasticity,
		Static:       spec.Static,
		AlignTopLeft: spec.AlignTopLeft,
		OffsetX:      spec.OffsetX,
		OffsetY:      spec.OffsetY,
	})
}

type gravityScaleSpec = prefabs.GravityScaleComponentSpec

func addGravityScale(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[gravityScaleSpec](raw)
	if err != nil {
		return fmt.Errorf("decode gravity scale spec: %w", err)
	}
	return ecs.Add(w, e, component.GravityScaleComponent.Kind(), &component.GravityScale{Scale: spec.Scale})
}

type hazardSpec = prefabs.HazardComponentSpec

func addHazard(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[hazardSpec](raw)
	if err != nil {
		return fmt.Errorf("decode hazard spec: %w", err)
	}
	width := spec.Width
	height := spec.Height
	if (width <= 0 || height <= 0) && spec.AutoSizeFromSprite {
		if s, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && s != nil && s.Image != nil {
			iw, ih := s.Image.Size()
			if width <= 0 {
				width = float64(iw)
			}
			if height <= 0 {
				height = float64(ih)
			}
		}
	}
	if spec.ScaleWithTransform {
		if tr, ok := ecs.Get(w, e, component.TransformComponent.Kind()); ok && tr != nil {
			width *= tr.ScaleX
			height *= tr.ScaleY
		}
	}
	if width <= 0 {
		width = 32
	}
	if height <= 0 {
		height = 32
	}
	return ecs.Add(w, e, component.HazardComponent.Kind(), &component.Hazard{
		Width:   width,
		Height:  height,
		OffsetX: spec.OffsetX,
		OffsetY: spec.OffsetY,
	})
}

type healthSpec = prefabs.HealthComponentSpec

func addHealth(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[healthSpec](raw)
	if err != nil {
		return fmt.Errorf("decode health spec: %w", err)
	}
	if spec.Initial == 0 {
		spec.Initial = 1
	}
	if spec.Current == 0 {
		spec.Current = spec.Initial
	}
	return ecs.Add(w, e, component.HealthComponent.Kind(), &component.Health{Initial: spec.Initial, Current: spec.Current})
}

type hitboxSpec = prefabs.HitboxComponentSpec

func addHitboxes(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[[]hitboxSpec](raw)
	if err != nil {
		return fmt.Errorf("decode hitbox spec: %w", err)
	}
	if len(spec) == 0 {
		return nil
	}
	sx, sy := 1.0, 1.0
	if tr, ok := ecs.Get(w, e, component.TransformComponent.Kind()); ok && tr != nil {
		sx, sy = tr.ScaleX, tr.ScaleY
	}
	out := make([]component.Hitbox, 0, len(spec))
	for _, hb := range spec {
		out = append(out, component.Hitbox{
			Width:   hb.Width * sx,
			Height:  hb.Height * sy,
			OffsetX: hb.OffsetX,
			OffsetY: hb.OffsetY,
			Damage:  hb.Damage,
			Anim:    hb.Anim,
			Frames:  hb.Frames,
		})
	}
	return ecs.Add(w, e, component.HitboxComponent.Kind(), &out)
}

type collisionLayerSpec = prefabs.CollisionLayerComponentSpec

func addCollisionLayer(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[collisionLayerSpec](raw)
	if err != nil {
		return fmt.Errorf("decode collision layer spec: %w", err)
	}
	cat := spec.Category
	mask := spec.Mask
	if cat == 0 {
		cat = 1
	}
	if mask == 0 {
		mask = ^uint32(0)
	}
	return ecs.Add(w, e, component.CollisionLayerComponent.Kind(), &component.CollisionLayer{Category: cat, Mask: mask})
}

type repulsionLayerSpec = prefabs.RepulsionLayerComponentSpec

func addRepulsionLayer(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[repulsionLayerSpec](raw)
	if err != nil {
		return fmt.Errorf("decode repulsion layer spec: %w", err)
	}
	cat := spec.Category
	mask := spec.Mask
	if cat == 0 {
		cat = 1
	}
	if mask == 0 {
		mask = ^uint32(0)
	}
	return ecs.Add(w, e, component.RepulsionLayerComponent.Kind(), &component.RepulsionLayer{Category: cat, Mask: mask})
}

type hurtboxSpec = prefabs.HurtboxComponentSpec

func addHurtboxes(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[[]hurtboxSpec](raw)
	if err != nil {
		return fmt.Errorf("decode hurtbox spec: %w", err)
	}
	if len(spec) == 0 {
		return nil
	}
	sx, sy := 1.0, 1.0
	if tr, ok := ecs.Get(w, e, component.TransformComponent.Kind()); ok && tr != nil {
		sx, sy = tr.ScaleX, tr.ScaleY
	}
	out := make([]component.Hurtbox, 0, len(spec))
	for _, hb := range spec {
		out = append(out, component.Hurtbox{
			Width:   hb.Width * sx,
			Height:  hb.Height * sy,
			OffsetX: hb.OffsetX,
			OffsetY: hb.OffsetY,
		})
	}
	return ecs.Add(w, e, component.HurtboxComponent.Kind(), &out)
}

func addAINavigation(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.AINavigationComponent.Kind(), &component.AINavigation{})
}

type anchorSpec = prefabs.AnchorComponentSpec

func addAnchor(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[anchorSpec](raw)
	if err != nil {
		return fmt.Errorf("decode anchor spec: %w", err)
	}
	return ecs.Add(w, e, component.AnchorComponent.Kind(), &component.Anchor{TargetX: spec.TargetX, TargetY: spec.TargetY, Speed: spec.Speed})
}

func addKnockbackable(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	return ecs.Add(w, e, component.KnockbackableComponent.Kind(), &component.Knockbackable{})
}

func buildAudioComponentFromSpec(audioSpecs []audioClipSpec) (*component.Audio, error) {
	n := len(audioSpecs)
	if n == 0 {
		return nil, nil
	}

	names := make([]string, 0, n)
	players := make([]*audio.Player, 0, n)
	volume := make([]float64, 0, n)
	play := make([]bool, 0, n)
	stop := make([]bool, 0, n)

	for i, clip := range audioSpecs {
		player, err := assets.LoadAudioPlayer(clip.File)
		if err != nil {
			return nil, fmt.Errorf("audio clip %d (%q): %w", i, clip.Name, err)
		}
		names = append(names, clip.Name)
		players = append(players, player)
		volume = append(volume, clip.Volume)
		play = append(play, false)
		stop = append(stop, false)
	}

	return &component.Audio{
		Names:   names,
		Players: players,
		Volume:  volume,
		Play:    play,
		Stop:    stop,
	}, nil
}

func parseHexColor(v string) (color.Color, error) {
	s := strings.TrimPrefix(strings.TrimSpace(v), "#")
	if len(s) != 6 && len(s) != 8 {
		return nil, fmt.Errorf("invalid color format: %q", v)
	}
	parse := func(start int) (uint8, error) {
		n, err := strconv.ParseUint(s[start:start+2], 16, 8)
		return uint8(n), err
	}
	r, err := parse(0)
	if err != nil {
		return nil, fmt.Errorf("parse red component: %w", err)
	}
	g, err := parse(2)
	if err != nil {
		return nil, fmt.Errorf("parse green component: %w", err)
	}
	b, err := parse(4)
	if err != nil {
		return nil, fmt.Errorf("parse blue component: %w", err)
	}
	a := uint8(255)
	if len(s) == 8 {
		a, err = parse(6)
		if err != nil {
			return nil, fmt.Errorf("parse alpha component: %w", err)
		}
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}
