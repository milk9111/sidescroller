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

	entity := w.CreateEntity()

	if err := ecs.Add(w, entity, component.PlayerTagComponent, component.PlayerTag{}); err != nil {
		return 0, fmt.Errorf("player: add player tag: %w", err)
	}

	if err := ecs.Add(w, entity, component.PlayerComponent, component.Player{
		MoveSpeed:        playerSpec.MoveSpeed,
		JumpSpeed:        playerSpec.JumpSpeed,
		JumpHoldFrames:   playerSpec.JumpHoldFrames,
		JumpHoldBoost:    playerSpec.JumpHoldBoost,
		CoyoteFrames:     playerSpec.CoyoteFrames,
		WallGrabFrames:   playerSpec.WallGrabFrames,
		WallSlideSpeed:   playerSpec.WallSlideSpeed,
		WallJumpPush:     playerSpec.WallJumpPush,
		WallJumpFrames:   playerSpec.WallJumpFrames,
		JumpBufferFrames: playerSpec.JumpBufferFrames,
	}); err != nil {
		return 0, fmt.Errorf("player: add player component: %w", err)
	}

	if err := ecs.Add(w, entity, component.InputComponent, component.Input{}); err != nil {
		return 0, fmt.Errorf("player: add input: %w", err)
	}

	if err := ecs.Add(w, entity, component.PlayerStateMachineComponent, component.PlayerStateMachine{}); err != nil {
		return 0, fmt.Errorf("player: add state machine: %w", err)
	}

	// add collision state component used by the physics handlers
	if err := ecs.Add(w, entity, component.PlayerCollisionComponent, component.PlayerCollision{}); err != nil {
		return 0, fmt.Errorf("player: add collision component: %w", err)
	}

	playerTransform := component.Transform{
		X:        playerSpec.Transform.X,
		Y:        playerSpec.Transform.Y,
		ScaleX:   playerSpec.Transform.ScaleX,
		ScaleY:   playerSpec.Transform.ScaleY,
		Rotation: playerSpec.Transform.Rotation,
	}
	if err := ecs.Add(
		w,
		entity,
		component.TransformComponent,
		playerTransform,
	); err != nil {
		return 0, fmt.Errorf("player: add transform: %w", err)
	}

	if err := ecs.Add(
		w,
		entity,
		component.RenderLayerComponent,
		component.RenderLayer{Index: playerSpec.RenderLayer.Index},
	); err != nil {
		return 0, fmt.Errorf("player: add render layer: %w", err)
	}

	if err := ecs.Add(
		w,
		entity,
		component.SpriteComponent,
		component.Sprite{
			UseSource: playerSpec.Sprite.UseSource,
			OriginX:   playerSpec.Sprite.OriginX,
			OriginY:   playerSpec.Sprite.OriginY,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add sprite: %w", err)
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
		component.AnimationComponent,
		component.Animation{
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

	// apply collider offsets from spec (scaled by transform)
	offsetX := playerSpec.Collider.OffsetX * playerTransform.ScaleX
	offsetY := playerSpec.Collider.OffsetY * playerTransform.ScaleY

	if err := ecs.Add(
		w,
		entity,
		component.PhysicsBodyComponent,
		component.PhysicsBody{
			Width:        width,
			Height:       height,
			Mass:         1,
			Friction:     0,
			Elasticity:   0,
			AlignTopLeft: true,
			OffsetX:      offsetX,
			OffsetY:      offsetY,
		},
	); err != nil {
		return 0, fmt.Errorf("player: add physics body: %w", err)
	}

	return entity, nil
}

func NewPlayerAt(w *ecs.World, x, y float64) (ecs.Entity, error) {
	entity, err := NewPlayer(w)
	if err != nil {
		return 0, err
	}
	transform, ok := ecs.Get(w, entity, component.TransformComponent)
	if !ok {
		transform = component.Transform{ScaleX: 1, ScaleY: 1}
	}
	transform.X = x
	transform.Y = y
	if err := ecs.Add(w, entity, component.TransformComponent, transform); err != nil {
		return 0, fmt.Errorf("player: override transform: %w", err)
	}
	return entity, nil
}
