package system

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestShrinePopupSystemShowsClosestShrine(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 62, Y: 122, ScaleX: 1, ScaleY: 1})

	uiRoot := ecs.CreateEntity(w)
	_ = ecs.Add(w, uiRoot, component.DialogueInputComponent.Kind(), &component.DialogueInput{})

	popup := ecs.CreateEntity(w)
	_ = ecs.Add(w, popup, component.ShrinePopupComponent.Kind(), &component.ShrinePopup{})
	_ = ecs.Add(w, popup, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: true, Image: ebiten.NewImage(16, 8)})

	near := ecs.CreateEntity(w)
	_ = ecs.Add(w, near, component.ShrineComponent.Kind(), &component.Shrine{Range: 100})
	_ = ecs.Add(w, near, component.TransformComponent.Kind(), &component.Transform{X: 60, Y: 120, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, near, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(128, 128), OriginX: 64, OriginY: 128})

	far := ecs.CreateEntity(w)
	_ = ecs.Add(w, far, component.ShrineComponent.Kind(), &component.Shrine{Range: 100})
	_ = ecs.Add(w, far, component.TransformComponent.Kind(), &component.Transform{X: 100, Y: 120, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, far, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(128, 128), OriginX: 64, OriginY: 128})

	NewShrinePopupSystem().Update(w)

	popupSprite, _ := ecs.Get(w, popup, component.SpriteComponent.Kind())
	if popupSprite == nil || popupSprite.Disabled {
		t.Fatal("expected shrine popup sprite to be enabled")
	}

	popupState, _ := ecs.Get(w, popup, component.ShrinePopupComponent.Kind())
	if popupState == nil || popupState.TargetShrineEntity != uint64(near) {
		t.Fatalf("expected near shrine target, got %+v", popupState)
	}

	popupTransform, _ := ecs.Get(w, popup, component.TransformComponent.Kind())
	if popupTransform == nil || popupTransform.X != 60 || popupTransform.Y != -14 {
		t.Fatalf("expected popup above shrine top center (60,-14), got %+v", popupTransform)
	}
}

func TestShrineSystemQueuesShrineHealRequest(t *testing.T) {
	w := ecs.NewWorld()

	level := ecs.CreateEntity(w)
	_ = ecs.Add(w, level, component.LevelRuntimeComponent.Kind(), &component.LevelRuntime{Name: "disposal_1.json"})

	player := ecs.CreateEntity(w)
	_ = ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{})
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 40, Y: 56, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, player, component.SpriteComponent.Kind(), &component.Sprite{FacingLeft: true})
	_ = ecs.Add(w, player, component.HealthComponent.Kind(), &component.Health{Initial: 5, Current: 1})
	_ = ecs.Add(w, player, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{HealUses: 2})

	input := ecs.CreateEntity(w)
	_ = ecs.Add(w, input, component.DialogueInputComponent.Kind(), &component.DialogueInput{Pressed: true})

	shrine := ecs.CreateEntity(w)
	_ = ecs.Add(w, shrine, component.ShrineComponent.Kind(), &component.Shrine{Range: 128})

	popup := ecs.CreateEntity(w)
	_ = ecs.Add(w, popup, component.ShrinePopupComponent.Kind(), &component.ShrinePopup{TargetShrineEntity: uint64(shrine)})
	_ = ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(16, 8)})

	NewShrineSystem().Update(w)

	health, _ := ecs.Get(w, player, component.HealthComponent.Kind())
	if health == nil || health.Current != 1 {
		t.Fatalf("expected shrine interaction to defer health reset until animation completes, got %+v", health)
	}

	stateMachine, _ := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind())
	if stateMachine == nil || stateMachine.HealUses != 2 {
		t.Fatalf("expected shrine interaction to defer flask reset until animation completes, got %+v", stateMachine)
	}

	if ecs.Has(w, player, component.PlayerCheckpointComponent.Kind()) {
		t.Fatal("expected checkpoint update to wait for shrine animation completion")
	}

	if ecs.Has(w, player, component.SafeRespawnComponent.Kind()) {
		t.Fatal("expected safe respawn update to wait for shrine animation completion")
	}

	req, ok := ecs.Get(w, player, component.ShrineHealRequestComponent.Kind())
	if !ok || req == nil {
		t.Fatal("expected shrine heal request on the player")
	}

	if _, ok := ecs.First(w, component.EnemyRespawnRequestComponent.Kind()); ok {
		t.Fatal("expected enemy respawn request to wait for shrine animation completion")
	}
}
