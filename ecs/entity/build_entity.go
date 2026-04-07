package entity

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/d5/tengo/v2"
	"github.com/d5/tengo/v2/stdlib"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

type entityPrefabSpec = prefabs.EntityBuildSpec

type buildContext struct {
	PrefabPath         string
	CenterSpriteOrigin bool
}

type componentBuildFn func(w *ecs.World, e ecs.Entity, raw any, ctx *buildContext) error

var componentRegistry = map[string]componentBuildFn{
	"player_tag":           addPlayerTag,
	"persistent":           addPersistent,
	"camera_tag":           addCameraTag,
	"aim_target_tag":       addAimTargetTag,
	"anchor_tag":           addAnchorTag,
	"spike_tag":            addSpikeTag,
	"ai_tag":               addAITag,
	"ai_phase_controller":  addAIPhaseController,
	"ai_phase_runtime":     addAIPhaseRuntime,
	"arena_node":           addArenaNode,
	"gate":                 addGate,
	"player":               addPlayer,
	"input":                addInput,
	"spawn_children":       addSpawnChildren,
	"player_state_machine": addPlayerStateMachine,
	"player_collision":     addPlayerCollision,
	"transform":            addTransform,
	"area_bounds":          addAreaBounds,
	"area_tile_stamp":      addAreaTileStamp,
	"parallax":             addParallax,
	"color":                addColor,
	"sprite":               addSprite,
	"render_layer":         addRenderLayer,
	"line_render":          addLineRender,
	"circle_render":        addCircleRender,
	"camera":               addCamera,
	"ai":                   addAI,
	"pathfinding":          addPathfinding,
	"ai_state":             addAIState,
	"ai_context":           addAIContext,
	"ai_config":            addAIConfig,
	"script":               addScript,
	"trigger":              addTrigger,
	"animation":            addAnimation,
	"audio":                addAudio,
	"music_player":         addMusicPlayer,
	"collision_layer":      addCollisionLayer,
	"repulsion_layer":      addRepulsionLayer,
	"physics_body":         addPhysicsBody,
	"gravity_scale":        addGravityScale,
	"hazard":               addHazard,
	"health":               addHealth,
	"breakable_wall":       addBreakableWall,
	"hitboxes":             addHitboxes,
	"hurtboxes":            addHurtboxes,
	"ai_navigation":        addAINavigation,
	"anchor":               addAnchor,
	"pickup":               addPickup,
	"knockbackable":        addKnockbackable,
	"ttl":                  addTTL,
	"sprite_shake":         addSpriteShake,
	"sprite_fade_out":      addSpriteFadeOut,
	"inventory":            addInventory,
	"item_reference":       addItemReference,
	"item":                 addItem,
	"dialogue":             addDialogue,
	"item_popup":           addItemPopup,
	"dialogue_popup":       addDialoguePopup,
	"particle_emitter":     addParticleEmitter,
}

var componentBuildOrder = []string{
	"player_tag",
	"persistent",
	"camera_tag",
	"aim_target_tag",
	"anchor_tag",
	"spike_tag",
	"ai_tag",
	"arena_node",
	"gate",
	"player",
	"input",
	"spawn_children",
	"player_state_machine",
	"player_collision",
	"transform",
	"area_bounds",
	"area_tile_stamp",
	"parallax",
	"color",
	"sprite",
	"render_layer",
	"line_render",
	"circle_render",
	"camera",
	"ai",
	"pathfinding",
	"ai_state",
	"ai_context",
	"ai_config",
	"script",
	"trigger",
	"ai_phase_controller",
	"ai_phase_runtime",
	"animation",
	"audio",
	"music_player",
	"collision_layer",
	"repulsion_layer",
	"physics_body",
	"gravity_scale",
	"hazard",
	"health",
	"breakable_wall",
	"hitboxes",
	"hurtboxes",
	"ai_navigation",
	"anchor",
	"pickup",
	"knockbackable",
	"ttl",
	"sprite_shake",
	"sprite_fade_out",
	"inventory",
	"item_reference",
	"item",
	"dialogue",
	"item_popup",
	"dialogue_popup",
	"particle_emitter",
}

func BuildEntity(w *ecs.World, prefabPath string) (ecs.Entity, error) {
	return BuildEntityWithOverrides(w, prefabPath, nil)
}

func BuildEntityWithOverrides(w *ecs.World, prefabPath string, componentOverrides map[string]any) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("build entity: world is nil")
	}

	spec, err := prefabs.LoadEntityBuildSpecWithOverrides(prefabPath, componentOverrides)
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

func addParticleEmitter(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.ParticleEmitterComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode particle_emitter spec: %w", err)
	}

	var img *ebiten.Image
	if spec.Image != "" {
		img, err = assets.LoadImage(spec.Image)
		if err != nil {
			return fmt.Errorf("decode particle_emitter spec: load image %q: %w", spec.Image, err)
		}
	}

	if spec.Color != "" {
		c := component.Color{R: 1, G: 1, B: 1, A: 1}
		parsed, err := parseHexColor(spec.Color)
		if err != nil {
			return fmt.Errorf("parse color hex: %w", err)
		}
		nrgba := color.NRGBAModel.Convert(parsed).(color.NRGBA)
		c.R = float64(nrgba.R) / 255.0
		c.G = float64(nrgba.G) / 255.0
		c.B = float64(nrgba.B) / 255.0
		c.A = float64(nrgba.A) / 255.0

		err = ecs.Add(w, e, component.ColorComponent.Kind(), &c)
		if err != nil {
			return fmt.Errorf("decode particle_emitter spec: add color component: %w", err)
		}
	}

	emitter := &component.ParticleEmitter{
		Name:     spec.Name,
		Disabled: spec.Disabled,
		Pool: sync.Pool{
			New: func() any {
				return &component.Particle{}
			},
		},
		Particles:      make([]*component.Particle, 0, spec.TotalParticles),
		TotalParticles: spec.TotalParticles,
		Lifetime:       spec.Lifetime,
		Burst:          spec.Burst,
		HasGravity:     spec.HasGravity,
		Continuous:     spec.Continuous,
		Image:          img,
		Scale: struct {
			X float64
			Y float64
		}{
			X: spec.Scale.X,
			Y: spec.Scale.Y,
		},
	}

	// Pre-populate the pool with the total number of particles to avoid GC churn during gameplay
	for i := 0; i < spec.TotalParticles; i++ {
		p := emitter.Pool.Get().(*component.Particle)
		emitter.Pool.Put(p)
	}

	return ecs.Add(w, e, component.ParticleEmitterComponent.Kind(), emitter)
}

func addDialogue(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.DialogueComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode dialogue spec: %w", err)
	}

	lines := append([]string(nil), spec.Lines...)

	var portrait *ebiten.Image
	if spec.Portrait != "" {
		img, err := assets.LoadImage(spec.Portrait)
		if err != nil {
			return fmt.Errorf("decode dialogue spec: load portrait image %q: %w", spec.Portrait, err)
		}
		portrait = scaleImage(img, 3)
	}

	return ecs.Add(w, e, component.DialogueComponent.Kind(), &component.Dialogue{
		Lines:    lines,
		Range:    spec.Range,
		Portrait: portrait,
	})
}

func addItem(w *ecs.World, e ecs.Entity, raw any, ctx *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.ItemComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode item spec: %w", err)
	}

	var image *ebiten.Image
	if spec.Image != "" {
		img, err := assets.LoadImage(spec.Image)
		if err != nil {
			return fmt.Errorf("decode item spec: load image %q: %w", spec.Image, err)
		}
		image = scaleImage(img, 4)
	}

	return ecs.Add(w, e, component.ItemComponent.Kind(), &component.Item{
		Prefab:      ctx.PrefabPath,
		Description: spec.Description,
		Range:       spec.Range,
		Image:       image,
	})
}

func addItemReference(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.ItemReferenceComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode item_reference spec: %w", err)
	}

	prefabPath := strings.TrimSpace(spec.Prefab)

	return ecs.Add(w, e, component.ItemReferenceComponent.Kind(), &component.ItemReference{Prefab: prefabPath})
}

func addInventory(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.InventoryComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode inventory spec: %w", err)
	}

	items := make([]component.InventoryItem, 0, len(spec.Items))
	for _, entry := range spec.Items {
		if strings.TrimSpace(entry.Prefab) == "" {
			continue
		}
		count := entry.Count
		if count <= 0 {
			count = 1
		}
		items = append(items, component.InventoryItem{
			Prefab: entry.Prefab,
			Count:  count,
		})
	}

	return ecs.Add(w, e, component.InventoryComponent.Kind(), &component.Inventory{Items: items})
}

func scaleImage(src *ebiten.Image, factor float64) *ebiten.Image {
	if src == nil || factor <= 0 || factor == 1 {
		return src
	}
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return src
	}
	scaledWidth := int(float64(width) * factor)
	scaledHeight := int(float64(height) * factor)
	if scaledWidth <= 0 || scaledHeight <= 0 {
		return src
	}
	dst := ebiten.NewImage(scaledWidth, scaledHeight)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(factor, factor)
	dst.DrawImage(src, op)
	return dst
}

func addDialoguePopup(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.DialoguePopupComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode dialogue_popup spec: %w", err)
	}

	var keyboardCue *ebiten.Image
	if spec.KeyboardCue != "" {
		img, err := assets.LoadImage(spec.KeyboardCue)
		if err != nil {
			return fmt.Errorf("decode dialogue_popup spec:load image %q: %w", spec.KeyboardCue, err)
		}
		keyboardCue = img
	}

	var gamepadCue *ebiten.Image
	if spec.GamepadCue != "" {
		img, err := assets.LoadImage(spec.GamepadCue)
		if err != nil {
			return fmt.Errorf("decode dialogue_popup spec:load image %q: %w", spec.GamepadCue, err)
		}
		gamepadCue = img
	}

	var base *ebiten.Image
	if spec.Base != "" {
		img, err := assets.LoadImage(spec.Base)
		if err != nil {
			return fmt.Errorf("decode dialogue_popup spec:load image %q: %w", spec.Base, err)
		}
		base = img
	}

	return ecs.Add(w, e, component.DialoguePopupComponent.Kind(), &component.DialoguePopup{
		KeyboardCue: keyboardCue,
		GamepadCue:  gamepadCue,
		Base:        base,
	})
}

func addItemPopup(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.ItemPopupComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode item_popup spec: %w", err)
	}

	var keyboardCue *ebiten.Image
	if spec.KeyboardCue != "" {
		img, err := assets.LoadImage(spec.KeyboardCue)
		if err != nil {
			return fmt.Errorf("decode item_popup spec:load image %q: %w", spec.KeyboardCue, err)
		}
		keyboardCue = img
	}

	var gamepadCue *ebiten.Image
	if spec.GamepadCue != "" {
		img, err := assets.LoadImage(spec.GamepadCue)
		if err != nil {
			return fmt.Errorf("decode item_popup spec:load image %q: %w", spec.GamepadCue, err)
		}
		gamepadCue = img
	}

	var base *ebiten.Image
	if spec.Base != "" {
		img, err := assets.LoadImage(spec.Base)
		if err != nil {
			return fmt.Errorf("decode item_popup spec:load image %q: %w", spec.Base, err)
		}
		base = img
	}

	return ecs.Add(w, e, component.ItemPopupComponent.Kind(), &component.ItemPopup{
		KeyboardCue: keyboardCue,
		GamepadCue:  gamepadCue,
		Base:        base,
	})
}

func addPlayerTag(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
}

func addPersistent(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	id := ""
	keepOnLevelChange := true
	keepOnReload := false

	if rawMap, ok := raw.(map[string]any); ok {
		if value, ok := rawMap["id"].(string); ok {
			id = value
		}
		if value, ok := rawMap["keep_on_level_change"].(bool); ok {
			keepOnLevelChange = value
		}
		if value, ok := rawMap["keep_on_reload"].(bool); ok {
			keepOnReload = value
		}
	}

	return ecs.Add(w, e, component.PersistentComponent.Kind(), &component.Persistent{
		ID:                id,
		KeepOnLevelChange: keepOnLevelChange,
		KeepOnReload:      keepOnReload,
	})
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

func addGate(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	if err := ecs.Add(w, e, component.GateComponent.Kind(), &component.Gate{}); err != nil {
		return err
	}
	if !ecs.Has(w, e, component.GateRuntimeComponent.Kind()) {
		if err := ecs.Add(w, e, component.GateRuntimeComponent.Kind(), &component.GateRuntime{}); err != nil {
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
		FallMultiplier:       spec.FallMultiplier,
		CoyoteFrames:         spec.CoyoteFrames,
		WallGrabFrames:       spec.WallGrabFrames,
		WallSlideSpeed:       spec.WallSlideSpeed,
		WallJumpPush:         spec.WallJumpPush,
		WallJumpFrames:       spec.WallJumpFrames,
		JumpBufferFrames:     spec.JumpBufferFrames,
		AnchorReelSpeed:      spec.AnchorReelSpeed,
		AnchorMinLength:      spec.AnchorMinLength,
		AimSlowFactor:        spec.AimSlowFactor,
		HitFreezeFrames:      spec.HitFreezeFrames,
		DamageShakeIntensity: spec.DamageShakeIntensity,
	})
}

func addInput(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.InputComponent.Kind(), &component.Input{})
}

type spawnChildrenSpec = prefabs.SpawnChildrenComponentSpec

func addSpawnChildren(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[spawnChildrenSpec](raw)
	if err != nil {
		return fmt.Errorf("decode spawn_children spec: %w", err)
	}

	children := make([]component.SpawnChildSpec, 0, len(spec.Children))
	for _, child := range spec.Children {
		if child.Prefab == "" {
			continue
		}
		children = append(children, component.SpawnChildSpec{Prefab: child.Prefab})
	}

	if len(children) == 0 {
		return nil
	}

	if err := ecs.Add(w, e, component.SpawnChildrenComponent.Kind(), &component.SpawnChildren{Children: children}); err != nil {
		return err
	}

	if !ecs.Has(w, e, component.SpawnChildrenRuntimeComponent.Kind()) {
		if err := ecs.Add(w, e, component.SpawnChildrenRuntimeComponent.Kind(), &component.SpawnChildrenRuntime{Spawned: map[string]uint64{}}); err != nil {
			return err
		}
	}

	return nil
}

func addPlayerStateMachine(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{})
}

func addPlayerCollision(w *ecs.World, e ecs.Entity, _ any, _ *buildContext) error {
	return ecs.Add(w, e, component.PlayerCollisionComponent.Kind(), &component.PlayerCollision{})
}

type transformSpec = prefabs.TransformComponentSpec

type areaBoundsSpec = prefabs.AreaBoundsComponentSpec

func addAreaBounds(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[areaBoundsSpec](raw)
	if err != nil {
		return fmt.Errorf("decode area_bounds spec: %w", err)
	}
	return ecs.Add(w, e, component.AreaBoundsComponent.Kind(), &component.AreaBounds{Bounds: component.AABB{
		X: spec.Bounds.X,
		Y: spec.Bounds.Y,
		W: spec.Bounds.W,
		H: spec.Bounds.H,
	}})
}

type areaTileStampSpec = prefabs.AreaTileStampComponentSpec

func addAreaTileStamp(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[areaTileStampSpec](raw)
	if err != nil {
		return fmt.Errorf("decode area_tile_stamp spec: %w", err)
	}
	mode := component.AreaTileStampRotationMode(strings.TrimSpace(spec.RotationMode))
	if mode == "" {
		mode = component.AreaTileStampRotationNone
	}
	overdrawMode := component.AreaTileStampOverdrawMode(strings.TrimSpace(spec.OverdrawMode))
	if overdrawMode == "" {
		overdrawMode = component.AreaTileStampOverdrawNone
	}
	playerFacingSide := component.AreaTileStampSide(strings.TrimSpace(spec.PlayerFacingSide))
	if playerFacingSide == "" {
		playerFacingSide = component.AreaTileStampSideNone
	}
	return ecs.Add(w, e, component.AreaTileStampComponent.Kind(), &component.AreaTileStamp{
		TileWidth:        spec.TileWidth,
		TileHeight:       spec.TileHeight,
		Overdraw:         spec.Overdraw,
		OverdrawMode:     overdrawMode,
		PlayerFacingSide: playerFacingSide,
		RotationMode:     mode,
		RotationOffset:   spec.RotationOffset,
	})
}

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
		Parent:   spec.Parent,
	})
}

type parallaxSpec = prefabs.ParallaxComponentSpec

func addParallax(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[parallaxSpec](raw)
	if err != nil {
		return fmt.Errorf("decode parallax spec: %w", err)
	}

	parallax := &component.Parallax{
		FactorX: spec.FactorX,
		FactorY: spec.FactorY,
	}
	if spec.CameraAnchorX != nil {
		parallax.AnchorCameraX = *spec.CameraAnchorX
		parallax.HasAnchorCameraX = true
	}
	if spec.CameraAnchorY != nil {
		parallax.AnchorCameraY = *spec.CameraAnchorY
		parallax.HasAnchorCameraY = true
	}

	return ecs.Add(w, e, component.ParallaxComponent.Kind(), parallax)
}

type colorSpec = prefabs.ColorComponentSpec

func addColor(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[colorSpec](raw)
	if err != nil {
		return fmt.Errorf("decode color spec: %w", err)
	}

	c := component.Color{R: 1, G: 1, B: 1, A: 1}
	if spec.Hex != "" {
		parsed, err := parseHexColor(spec.Hex)
		if err != nil {
			return fmt.Errorf("parse color hex: %w", err)
		}
		nrgba := color.NRGBAModel.Convert(parsed).(color.NRGBA)
		c.R = float64(nrgba.R) / 255.0
		c.G = float64(nrgba.G) / 255.0
		c.B = float64(nrgba.B) / 255.0
		c.A = float64(nrgba.A) / 255.0
	}

	if spec.R != nil {
		c.R = *spec.R
	}
	if spec.G != nil {
		c.G = *spec.G
	}
	if spec.B != nil {
		c.B = *spec.B
	}
	if spec.A != nil {
		c.A = *spec.A
	}

	return ecs.Add(w, e, component.ColorComponent.Kind(), &c)
}

type spriteSpec = prefabs.SpriteComponentSpec

func addSprite(w *ecs.World, e ecs.Entity, raw any, ctx *buildContext) error {
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

	sprite.Disabled = spec.Disabled
	sprite.UseSource = spec.UseSource
	if spec.UseSource && spec.SourceW > 0 && spec.SourceH > 0 {
		sprite.Source = image.Rect(spec.SourceX, spec.SourceY, spec.SourceX+spec.SourceW, spec.SourceY+spec.SourceH)
	}
	sprite.TileX = spec.TileX
	sprite.TileY = spec.TileY
	sprite.OriginX = spec.OriginX
	sprite.OriginY = spec.OriginY
	if ctx != nil {
		ctx.CenterSpriteOrigin = spec.CenterOriginIfZero
	}
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
		StartX:         spec.StartX,
		StartY:         spec.StartY,
		EndX:           spec.EndX,
		EndY:           spec.EndY,
		Width:          spec.Width,
		Color:          c,
		AntiAlias:      spec.AntiAlias,
		BehindEntities: spec.BehindEntities,
	})
}

type circleRenderSpec = prefabs.CircleRenderComponentSpec

func addCircleRender(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[circleRenderSpec](raw)
	if err != nil {
		return fmt.Errorf("decode circle render spec: %w", err)
	}
	if spec.Width <= 0 {
		spec.Width = 1
	}
	c := color.Color(color.RGBA{R: 255, A: 255})
	if spec.Color != "" {
		parsed, err := parseHexColor(spec.Color)
		if err != nil {
			return fmt.Errorf("parse circle render color: %w", err)
		}
		c = parsed
	}
	return ecs.Add(w, e, component.CircleRenderComponent.Kind(), &component.CircleRender{
		OffsetX:   spec.OffsetX,
		OffsetY:   spec.OffsetY,
		Radius:    spec.Radius,
		Width:     spec.Width,
		Color:     c,
		AntiAlias: spec.AntiAlias,
		Disabled:  spec.Disabled,
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

type scriptSpec = prefabs.ScriptComponentSpec

type triggerSpec = prefabs.TriggerComponentSpec

func addScript(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[scriptSpec](raw)
	if err != nil {
		return fmt.Errorf("decode script spec: %w", err)
	}
	// Support single `path` or multiple `paths` in prefab spec.
	paths := append([]string(nil), spec.Paths...)
	if spec.Path != "" {
		// Prepend legacy single path to keep ordering predictable.
		paths = append([]string{spec.Path}, paths...)
	}
	return ecs.Add(w, e, component.ScriptComponent.Kind(), &component.Script{Path: spec.Path, Paths: paths, Modules: append([]string(nil), spec.Modules...)})
}

func addTrigger(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[triggerSpec](raw)
	if err != nil {
		return fmt.Errorf("decode trigger spec: %w", err)
	}
	return ecs.Add(w, e, component.TriggerComponent.Kind(), &component.Trigger{
		Bounds: component.AABB{
			X: spec.Bounds.X,
			Y: spec.Bounds.Y,
			W: spec.Bounds.W,
			H: spec.Bounds.H,
		},
		Name:     spec.Name,
		Disabled: spec.Disabled,
	})
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

func addAnimation(w *ecs.World, e ecs.Entity, raw any, ctx *buildContext) error {
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

	if err := ecs.Add(w, e, component.AnimationComponent.Kind(), &component.Animation{
		Sheet:      sheet,
		Defs:       defs,
		Current:    spec.Current,
		Frame:      spec.Frame,
		FrameTimer: spec.FrameTimer,
		Playing:    playing,
	}); err != nil {
		return err
	}

	applyAnimationCenteredOrigin(w, e, ctx, spec)
	return nil
}

func applyAnimationCenteredOrigin(w *ecs.World, e ecs.Entity, ctx *buildContext, spec animationSpec) {
	if ctx == nil || !ctx.CenterSpriteOrigin {
		return
	}
	sprite, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
	if !ok || sprite == nil || sprite.OriginX != 0 || sprite.OriginY != 0 {
		return
	}
	frameW, frameH, ok := animationFrameSize(spec)
	if !ok {
		return
	}
	sprite.OriginX = float64(frameW) / 2
	sprite.OriginY = float64(frameH) / 2
}

func animationFrameSize(spec animationSpec) (int, int, bool) {
	if len(spec.Defs) == 0 {
		return 0, 0, false
	}
	if def, ok := spec.Defs[spec.Current]; ok && def.FrameW > 0 && def.FrameH > 0 {
		return def.FrameW, def.FrameH, true
	}
	names := make([]string, 0, len(spec.Defs))
	for name := range spec.Defs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		def := spec.Defs[name]
		if def.FrameW > 0 && def.FrameH > 0 {
			return def.FrameW, def.FrameH, true
		}
	}
	return 0, 0, false
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

type musicPlayerSpec = prefabs.MusicPlayerComponentSpec

func addMusicPlayer(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[musicPlayerSpec](raw)
	if err != nil {
		return fmt.Errorf("decode music player spec: %w", err)
	}

	players := make(map[string]*audio.Player, len(spec.Songs))
	trackVolumes := make(map[string]float64, len(spec.Songs))
	for i, song := range spec.Songs {
		track := strings.TrimSpace(song.Track)
		if track == "" {
			continue
		}
		player, err := assets.LoadAudioPlayer(track)
		if err != nil {
			return fmt.Errorf("music song %d (%q): %w", i, track, err)
		}
		volume := song.Volume
		if volume <= 0 {
			volume = 1
		}
		if volume > 1 {
			volume = 1
		}
		players[track] = player
		trackVolumes[track] = volume
	}

	loop := true
	if spec.Loop != nil {
		loop = *spec.Loop
	}

	startTrack := strings.TrimSpace(spec.StartTrack)
	pendingVolume := 0.0
	pendingActive := false
	if startTrack != "" {
		pendingVolume = trackVolumes[startTrack]
		if pendingVolume <= 0 {
			pendingVolume = 1
		}
		pendingActive = true
	}

	return ecs.Add(w, e, component.MusicPlayerComponent.Kind(), &component.MusicPlayer{
		Players:       players,
		TrackVolumes:  trackVolumes,
		PendingTrack:  startTrack,
		PendingVolume: pendingVolume,
		PendingLoop:   loop,
		PendingActive: pendingActive,
	})
}

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
		Disabled:               spec.Disabled,
		LockRotation:           spec.LockRotation,
		AutoSizeFromAreaBounds: spec.AutoSizeFromAreaBounds,
		Width:                  width,
		Height:                 height,
		Radius:                 spec.Radius,
		Mass:                   spec.Mass,
		Friction:               spec.Friction,
		Elasticity:             spec.Elasticity,
		Static:                 spec.Static,
		AlignTopLeft:           spec.AlignTopLeft,
		OffsetX:                spec.OffsetX,
		OffsetY:                spec.OffsetY,
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

type breakableWallSpec = prefabs.BreakableWallComponentSpec

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

func addBreakableWall(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[breakableWallSpec](raw)
	if err != nil {
		return fmt.Errorf("decode breakable wall spec: %w", err)
	}

	return ecs.Add(w, e, component.BreakableWallComponent.Kind(), &component.BreakableWall{
		LayerName:             spec.LayerName,
		DestroyedSignalTarget: spec.DestroyedSignalTarget,
	})
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

func addTTL(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.TTLComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode TTL spec: %w", err)
	}
	return ecs.Add(w, e, component.TTLComponent.Kind(), &component.TTL{Frames: spec.Frames})
}

func addSpriteShake(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.SpriteShakeComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode sprite shake spec: %w", err)
	}
	return ecs.Add(w, e, component.SpriteShakeComponent.Kind(), &component.SpriteShake{
		Frames:    spec.Frames,
		Intensity: spec.Intensity,
	})
}

func addSpriteFadeOut(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[prefabs.SpriteFadeOutComponentSpec](raw)
	if err != nil {
		return fmt.Errorf("decode sprite fade out spec: %w", err)
	}
	return ecs.Add(w, e, component.SpriteFadeOutComponent.Kind(), &component.SpriteFadeOut{
		Frames:      spec.Frames,
		TotalFrames: spec.Frames,
		Alpha:       1,
	})
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

type pickupSpec = prefabs.PickupComponentSpec

func addPickup(w *ecs.World, e ecs.Entity, raw any, _ *buildContext) error {
	spec, err := prefabs.DecodeComponentSpec[pickupSpec](raw)
	if err != nil {
		return fmt.Errorf("decode pickup spec: %w", err)
	}
	if spec.BobAmplitude == 0 {
		spec.BobAmplitude = 4
	}
	if spec.BobSpeed == 0 {
		spec.BobSpeed = 0.08
	}
	if spec.CollisionWidth == 0 {
		spec.CollisionWidth = 24
	}
	if spec.CollisionHeight == 0 {
		spec.CollisionHeight = 24
	}
	return ecs.Add(w, e, component.PickupComponent.Kind(), &component.Pickup{
		Kind:            spec.Kind,
		BobAmplitude:    spec.BobAmplitude,
		BobSpeed:        spec.BobSpeed,
		BobPhase:        spec.BobPhase,
		CollisionWidth:  spec.CollisionWidth,
		CollisionHeight: spec.CollisionHeight,
		GrantDoubleJump: spec.GrantDoubleJump,
		GrantWallGrab:   spec.GrantWallGrab,
		GrantAnchor:     spec.GrantAnchor,
	})
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
