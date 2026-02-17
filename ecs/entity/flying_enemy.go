package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewFlyingEnemy(w *ecs.World) (ecs.Entity, error) {
	spec, err := prefabs.LoadSpec[prefabs.EnemySpec]("flying_enemy.yaml")
	if err != nil {
		return 0, fmt.Errorf("flying enemy: load spec: %w", err)
	}

	entity := ecs.CreateEntity(w)

	if err := ecs.Add(w, entity, component.AITagComponent.Kind(), &component.AITag{}); err != nil {
		return 0, fmt.Errorf("flying enemy: add ai tag: %w", err)
	}

	if err := ecs.Add(w, entity, component.AIComponent.Kind(), &component.AI{
		MoveSpeed:    spec.MoveSpeed,
		FollowRange:  spec.FollowRange,
		AttackRange:  spec.AttackRange,
		AttackFrames: spec.AttackFrames,
	}); err != nil {
		return 0, fmt.Errorf("flying enemy: add ai component: %w", err)
	}

	if err := ecs.Add(w, entity, component.AIStateComponent.Kind(), &component.AIState{}); err != nil {
		return 0, fmt.Errorf("flying enemy: add ai state: %w", err)
	}
	if err := ecs.Add(w, entity, component.AIContextComponent.Kind(), &component.AIContext{}); err != nil {
		return 0, fmt.Errorf("flying enemy: add ai context: %w", err)
	}

	var specPtr *prefabs.FSMSpec
	if spec.FSM.Initial != "" || len(spec.FSM.States) > 0 {
		specPtr = &spec.FSM
	}
	if err := ecs.Add(w, entity, component.AIConfigComponent.Kind(), &component.AIConfig{FSM: "", Spec: specPtr}); err != nil {
		return 0, fmt.Errorf("flying enemy: add ai config: %w", err)
	}

	transform := &component.Transform{
		X:        spec.Transform.X,
		Y:        spec.Transform.Y,
		ScaleX:   spec.Transform.ScaleX,
		ScaleY:   spec.Transform.ScaleY,
		Rotation: spec.Transform.Rotation,
	}
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), transform); err != nil {
		return 0, fmt.Errorf("flying enemy: add transform: %w", err)
	}

	if err := ecs.Add(w, entity, component.SpriteComponent.Kind(), &component.Sprite{
		UseSource: spec.Sprite.UseSource,
		OriginX:   spec.Sprite.OriginX,
		OriginY:   spec.Sprite.OriginY,
	}); err != nil {
		return 0, fmt.Errorf("flying enemy: add sprite: %w", err)
	}

	if err := ecs.Add(w, entity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: spec.RenderLayer.Index}); err != nil {
		return 0, fmt.Errorf("flying enemy: add render layer: %w", err)
	}

	spriteSheet, err := assets.LoadImage(spec.Animation.Sheet)
	if err != nil {
		return 0, fmt.Errorf("flying enemy: load sprite sheet: %w", err)
	}

	defs := make(map[string]component.AnimationDef, len(spec.Animation.Defs))
	for name, defSpec := range spec.Animation.Defs {
		defs[name] = component.AnimationDef{
			Row:        defSpec.Row,
			ColStart:   defSpec.ColStart,
			FrameCount: defSpec.FrameCount,
			FrameW:     defSpec.FrameW,
			FrameH:     defSpec.FrameH,
			FPS:        defSpec.FPS,
			Loop:       defSpec.Loop,
		}
	}

	if err := ecs.Add(w, entity, component.AnimationComponent.Kind(), &component.Animation{
		Sheet:      spriteSheet,
		Defs:       defs,
		Current:    spec.Animation.Current,
		Frame:      0,
		FrameTimer: 0,
		Playing:    true,
	}); err != nil {
		return 0, fmt.Errorf("flying enemy: add animation: %w", err)
	}

	if len(spec.Audio) > 0 {
		audioComp, err := buildAudioComponent(spec.Audio)
		if err != nil {
			return 0, fmt.Errorf("flying enemy: build audio component: %w", err)
		}
		if audioComp != nil {
			if err := ecs.Add(w, entity, component.AudioComponent.Kind(), audioComp); err != nil {
				return 0, fmt.Errorf("flying enemy: add audio: %w", err)
			}
		}
	}

	width := spec.Collider.Width
	height := spec.Collider.Height
	if transform.ScaleX == 0 {
		transform.ScaleX = 1
	}
	if transform.ScaleY == 0 {
		transform.ScaleY = 1
	}
	width *= transform.ScaleX
	height *= transform.ScaleY
	if width == 0 {
		width = 32
	}
	if height == 0 {
		height = 32
	}

	if err := ecs.Add(w, entity, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{
		Width:        width,
		Height:       height,
		OffsetX:      spec.Collider.OffsetX,
		OffsetY:      spec.Collider.OffsetY,
		Mass:         1,
		Friction:     0.2,
		Elasticity:   0,
		AlignTopLeft: true,
	}); err != nil {
		return 0, fmt.Errorf("flying enemy: add physics body: %w", err)
	}

	if err := ecs.Add(w, entity, component.GravityScaleComponent.Kind(), &component.GravityScale{Scale: 0}); err != nil {
		return 0, fmt.Errorf("flying enemy: add gravity scale: %w", err)
	}

	// Mark enemy as a hazard so the player takes damage when walking into it.
	// Use the collider size/offset (already scaled) as the hazard bounds.
	hazardW := width
	hazardH := height
	if hazardW > 0 && hazardH > 0 {
		if err := ecs.Add(w, entity, component.HazardComponent.Kind(), &component.Hazard{
			Width:   hazardW,
			Height:  hazardH,
			OffsetX: spec.Collider.OffsetX,
			OffsetY: spec.Collider.OffsetY,
		}); err != nil {
			return 0, fmt.Errorf("flying enemy: add hazard: %w", err)
		}
	}

	hp := spec.Health
	if hp == 0 {
		hp = 5
	}
	if err := ecs.Add(w, entity, component.HealthComponent.Kind(), &component.Health{Initial: hp, Current: hp}); err != nil {
		return 0, fmt.Errorf("flying enemy: add health: %w", err)
	}

	if len(spec.Hitboxes) > 0 {
		hbs := make([]component.Hitbox, 0, len(spec.Hitboxes))
		for _, hb := range spec.Hitboxes {
			hbs = append(hbs, component.Hitbox{
				Width:   hb.Width * transform.ScaleX,
				Height:  hb.Height * transform.ScaleY,
				OffsetX: hb.OffsetX,
				OffsetY: hb.OffsetY,
				Damage:  hb.Damage,
				Anim:    hb.Anim,
				Frames:  hb.Frames,
			})
		}
		if err := ecs.Add(w, entity, component.HitboxComponent.Kind(), &hbs); err != nil {
			return 0, fmt.Errorf("flying enemy: add hitboxes: %w", err)
		}
	}

	if len(spec.Hurtboxes) > 0 {
		hbs := make([]component.Hurtbox, 0, len(spec.Hurtboxes))
		for _, hb := range spec.Hurtboxes {
			hbs = append(hbs, component.Hurtbox{
				Width:   hb.Width * transform.ScaleX,
				Height:  hb.Height * transform.ScaleY,
				OffsetX: hb.OffsetX,
				OffsetY: hb.OffsetY,
			})
		}
		if err := ecs.Add(w, entity, component.HurtboxComponent.Kind(), &hbs); err != nil {
			return 0, fmt.Errorf("flying enemy: add hurtboxes: %w", err)
		}
	}

	return entity, nil
}

func NewFlyingEnemyAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	entity, err := NewFlyingEnemy(w)
	if err != nil {
		return 0, err
	}
	transform, ok := ecs.Get(w, entity, component.TransformComponent.Kind())
	if !ok {
		transform = &component.Transform{ScaleX: 1, ScaleY: 1}
	}
	transform.X = x
	transform.Y = y
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), transform); err != nil {
		return 0, fmt.Errorf("flying enemy: override transform: %w", err)
	}
	return entity, nil
}
