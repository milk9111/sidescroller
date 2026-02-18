package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewPlayer(w *ecs.World) (ecs.Entity, error) {
	playerSpec, err := prefabs.LoadPlayerSpec()
	if err != nil {
		return 0, fmt.Errorf("player: load spec: %w", err)
	}

	entity := ecs.CreateEntity(w)

	if err := ecs.Add(w, entity, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		return 0, fmt.Errorf("player: add player tag: %w", err)
	}

	if err := ecs.Add(w, entity, component.PlayerComponent.Kind(), &component.Player{
		MoveSpeed:            playerSpec.MoveSpeed,
		JumpSpeed:            playerSpec.JumpSpeed,
		CoyoteFrames:         playerSpec.CoyoteFrames,
		WallSlideSpeed:       playerSpec.WallSlideSpeed,
		WallJumpPush:         playerSpec.WallJumpPush,
		WallJumpFrames:       playerSpec.WallJumpFrames,
		JumpBufferFrames:     playerSpec.JumpBufferFrames,
		JumpHoldFrames:       playerSpec.JumpHoldFrames,
		JumpHoldBoost:        playerSpec.JumpHoldBoost,
		AimSlowFactor:        playerSpec.AimSlowFactor,
		HitFreezeFrames:      playerSpec.HitFreezeFrames,
		DamageShakeIntensity: playerSpec.DamageShakeIntensity,
	}); err != nil {
		return 0, fmt.Errorf("player: add player component: %w", err)
	}

	if err := ecs.Add(w, entity, component.InputComponent.Kind(), &component.Input{}); err != nil {
		return 0, fmt.Errorf("player: add input: %w", err)
	}

	if err := ecs.Add(w, entity, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{}); err != nil {
		return 0, fmt.Errorf("player: add state machine: %w", err)
	}

	if err := ecs.Add(w, entity, component.PlayerCollisionComponent.Kind(), &component.PlayerCollision{}); err != nil {
		return 0, fmt.Errorf("player: add player collision: %w", err)
	}

	playerTransform := &component.Transform{
		X:        playerSpec.Transform.X,
		Y:        playerSpec.Transform.Y,
		ScaleX:   playerSpec.Transform.ScaleX,
		ScaleY:   playerSpec.Transform.ScaleY,
		Rotation: playerSpec.Transform.Rotation,
	}
	if err := ecs.Add(
		w,
		entity,
		component.TransformComponent.Kind(),
		playerTransform,
	); err != nil {
		return 0, fmt.Errorf("player: add transform: %w", err)
	}

	if err := ecs.Add(
		w,
		entity,
		component.SpriteComponent.Kind(),
		&component.Sprite{
			UseSource: playerSpec.Sprite.UseSource,
			OriginX:   playerSpec.Sprite.OriginX,
			OriginY:   playerSpec.Sprite.OriginY,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add sprite: %w", err)
	}

	if err := ecs.Add(w, entity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: playerSpec.RenderLayer.Index}); err != nil {
		return 0, fmt.Errorf("player: add render layer: %w", err)
	}

	spriteSheet, err := assets.LoadImage(playerSpec.Animation.Sheet)
	if err != nil {
		return 0, fmt.Errorf("player: load sprite sheet: %w", err)
	}

	defs := make(map[string]component.AnimationDef, len(playerSpec.Animation.Defs))
	for name, defSpec := range playerSpec.Animation.Defs {
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
			Current:    playerSpec.Animation.Current,
			Frame:      0,
			FrameTimer: 0,
			Playing:    true,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add animation: %w", err)
	}

	if len(playerSpec.Audio) > 0 {
		audioComp, err := buildAudioComponent(playerSpec.Audio)
		if err != nil {
			return 0, fmt.Errorf("player: build audio component: %w", err)
		}
		if audioComp != nil {
			if err := ecs.Add(w, entity, component.AudioComponent.Kind(), audioComp); err != nil {
				return 0, fmt.Errorf("player: add audio: %w", err)
			}
		}
	}

	width := playerSpec.Collider.Width
	height := playerSpec.Collider.Height

	if playerTransform.ScaleX == 0 {
		playerTransform.ScaleX = 1
	}
	if playerTransform.ScaleY == 0 {
		playerTransform.ScaleY = 1
	}

	width *= playerTransform.ScaleX
	height *= playerTransform.ScaleY
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
			OffsetX:      playerSpec.Collider.OffsetX,
			OffsetY:      playerSpec.Collider.OffsetY,
			Mass:         1,
			Friction:     0,
			Elasticity:   0,
			AlignTopLeft: true,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add physics body: %w", err)
	}

	// Health
	hp := playerSpec.Health
	if hp == 0 {
		hp = 10
	}
	if err := ecs.Add(w, entity, component.HealthComponent.Kind(), &component.Health{Initial: hp, Current: hp}); err != nil {
		return 0, fmt.Errorf("player: add health: %w", err)
	}

	// Hitboxes
	if len(playerSpec.Hitboxes) > 0 {
		hbs := make([]component.Hitbox, 0, len(playerSpec.Hitboxes))
		for _, hb := range playerSpec.Hitboxes {
			hbs = append(hbs, component.Hitbox{
				Width:   hb.Width * playerTransform.ScaleX,
				Height:  hb.Height * playerTransform.ScaleY,
				OffsetX: hb.OffsetX,
				OffsetY: hb.OffsetY,
				Damage:  hb.Damage,
				Anim:    hb.Anim,
				Frames:  hb.Frames,
			})
		}
		if err := ecs.Add(w, entity, component.HitboxComponent.Kind(), &hbs); err != nil {
			return 0, fmt.Errorf("player: add hitboxes: %w", err)
		}
	}

	// Hurtboxes
	if len(playerSpec.Hurtboxes) > 0 {
		hbs := make([]component.Hurtbox, 0, len(playerSpec.Hurtboxes))
		for _, hb := range playerSpec.Hurtboxes {
			hbs = append(hbs, component.Hurtbox{
				Width:   hb.Width * playerTransform.ScaleX,
				Height:  hb.Height * playerTransform.ScaleY,
				OffsetX: hb.OffsetX,
				OffsetY: hb.OffsetY,
			})
		}
		if err := ecs.Add(w, entity, component.HurtboxComponent.Kind(), &hbs); err != nil {
			return 0, fmt.Errorf("player: add hurtboxes: %w", err)
		}
	}

	return entity, nil
}

func NewPlayerAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	entity, err := NewPlayer(w)
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
		return 0, fmt.Errorf("player: override transform: %w", err)
	}
	return entity, nil
}
