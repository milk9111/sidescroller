package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewEnemy(w *ecs.World) (ecs.Entity, error) {
	enemySpec, err := prefabs.LoadEnemySpec()
	if err != nil {
		return 0, fmt.Errorf("enemy: load spec: %w", err)
	}

	entity := ecs.CreateEntity(w)

	if err := ecs.Add(w, entity, component.AITagComponent.Kind(), &component.AITag{}); err != nil {
		return 0, fmt.Errorf("enemy: add enemy tag: %w", err)
	}

	if err := ecs.Add(w, entity, component.AIComponent.Kind(), &component.AI{
		MoveSpeed:    enemySpec.MoveSpeed,
		FollowRange:  enemySpec.FollowRange,
		AttackRange:  enemySpec.AttackRange,
		AttackFrames: enemySpec.AttackFrames,
	}); err != nil {
		return 0, fmt.Errorf("enemy: add enemy component: %w", err)
	}

	if err := ecs.Add(w, entity, component.PathfindingComponent.Kind(), &component.Pathfinding{
		GridSize:      32,
		RepathFrames:  15,
		DebugNodeSize: 3,
	}); err != nil {
		return 0, fmt.Errorf("enemy: add pathfinding: %w", err)
	}

	if err := ecs.Add(w, entity, component.AIStateComponent.Kind(), &component.AIState{}); err != nil {
		return 0, fmt.Errorf("enemy: add ai state: %w", err)
	}

	if err := ecs.Add(w, entity, component.AIContextComponent.Kind(), &component.AIContext{}); err != nil {
		return 0, fmt.Errorf("enemy: add ai context: %w", err)
	}

	var specPtr *prefabs.FSMSpec
	if enemySpec.FSM.Initial != "" || len(enemySpec.FSM.States) > 0 {
		specPtr = &enemySpec.FSM
	}
	if err := ecs.Add(w, entity, component.AIConfigComponent.Kind(), &component.AIConfig{FSM: "", Spec: specPtr}); err != nil {
		return 0, fmt.Errorf("enemy: add ai config: %w", err)
	}

	enemyTransform := &component.Transform{
		X:        enemySpec.Transform.X,
		Y:        enemySpec.Transform.Y,
		ScaleX:   enemySpec.Transform.ScaleX,
		ScaleY:   enemySpec.Transform.ScaleY,
		Rotation: enemySpec.Transform.Rotation,
	}
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), enemyTransform); err != nil {
		return 0, fmt.Errorf("enemy: add transform: %w", err)
	}

	if err := ecs.Add(
		w,
		entity,
		component.SpriteComponent.Kind(),
		&component.Sprite{
			UseSource: enemySpec.Sprite.UseSource,
			OriginX:   enemySpec.Sprite.OriginX,
			OriginY:   enemySpec.Sprite.OriginY,
		},
	); err != nil {
		return 0, fmt.Errorf("enemy: add sprite: %w", err)
	}

	if err := ecs.Add(w, entity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: enemySpec.RenderLayer.Index}); err != nil {
		return 0, fmt.Errorf("enemy: add render layer: %w", err)
	}

	spriteSheet, err := assets.LoadImage(enemySpec.Animation.Sheet)
	if err != nil {
		return 0, fmt.Errorf("enemy: load sprite sheet: %w", err)
	}

	defs := make(map[string]component.AnimationDef, len(enemySpec.Animation.Defs))
	for name, defSpec := range enemySpec.Animation.Defs {
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

	if err := ecs.Add(
		w,
		entity,
		component.AnimationComponent.Kind(),
		&component.Animation{
			Sheet:      spriteSheet,
			Defs:       defs,
			Current:    enemySpec.Animation.Current,
			Frame:      0,
			FrameTimer: 0,
			Playing:    true,
		},
	); err != nil {
		return 0, fmt.Errorf("enemy: add animation: %w", err)
	}

	if len(enemySpec.Audio) > 0 {
		audioComp, err := buildAudioComponent(enemySpec.Audio)
		if err != nil {
			return 0, fmt.Errorf("enemy: build audio component: %w", err)
		}
		if audioComp != nil {
			if err := ecs.Add(w, entity, component.AudioComponent.Kind(), audioComp); err != nil {
				return 0, fmt.Errorf("enemy: add audio: %w", err)
			}
		}
	}

	width := enemySpec.Collider.Width
	height := enemySpec.Collider.Height

	if enemyTransform.ScaleX == 0 {
		enemyTransform.ScaleX = 1
	}
	if enemyTransform.ScaleY == 0 {
		enemyTransform.ScaleY = 1
	}

	width *= enemyTransform.ScaleX
	height *= enemyTransform.ScaleY
	if width == 0 {
		width = 32
	}
	if height == 0 {
		height = 32
	}

	if err := ecs.Add(
		w,
		entity,
		component.PhysicsBodyComponent.Kind(),
		&component.PhysicsBody{
			Width:        width,
			Height:       height,
			OffsetX:      enemySpec.Collider.OffsetX,
			OffsetY:      enemySpec.Collider.OffsetY,
			Mass:         1,
			Friction:     0.2,
			Elasticity:   0,
			AlignTopLeft: true,
		},
	); err != nil {
		return 0, fmt.Errorf("enemy: add physics body: %w", err)
	}

	// Health
	hp := enemySpec.Health
	if hp == 0 {
		hp = 5
	}
	if err := ecs.Add(w, entity, component.HealthComponent.Kind(), &component.Health{Initial: hp, Current: hp}); err != nil {
		return 0, fmt.Errorf("enemy: add health: %w", err)
	}

	// Hitboxes
	if len(enemySpec.Hitboxes) > 0 {
		hbs := make([]component.Hitbox, 0, len(enemySpec.Hitboxes))
		for _, hb := range enemySpec.Hitboxes {
			hbs = append(hbs, component.Hitbox{
				Width:   hb.Width * enemyTransform.ScaleX,
				Height:  hb.Height * enemyTransform.ScaleY,
				OffsetX: hb.OffsetX,
				OffsetY: hb.OffsetY,
				Damage:  hb.Damage,
				Anim:    hb.Anim,
				Frames:  hb.Frames,
			})
		}
		if err := ecs.Add(w, entity, component.HitboxComponent.Kind(), &hbs); err != nil {
			return 0, fmt.Errorf("enemy: add hitboxes: %w", err)
		}
	}

	// Hurtboxes
	if len(enemySpec.Hurtboxes) > 0 {
		hbs := make([]component.Hurtbox, 0, len(enemySpec.Hurtboxes))
		for _, hb := range enemySpec.Hurtboxes {
			hbs = append(hbs, component.Hurtbox{
				Width:   hb.Width * enemyTransform.ScaleX,
				Height:  hb.Height * enemyTransform.ScaleY,
				OffsetX: hb.OffsetX,
				OffsetY: hb.OffsetY,
			})
		}
		if err := ecs.Add(w, entity, component.HurtboxComponent.Kind(), &hbs); err != nil {
			return 0, fmt.Errorf("enemy: add hurtboxes: %w", err)
		}
	}

	if err := ecs.Add(w, entity, component.AINavigationComponent.Kind(), &component.AINavigation{}); err != nil {
		return 0, fmt.Errorf("enemy: add ai navigation component: %w", err)
	}

	return entity, nil
}

func NewEnemyAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	entity, err := NewEnemy(w)
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
		return 0, fmt.Errorf("enemy: override transform: %w", err)
	}
	return entity, nil
}
