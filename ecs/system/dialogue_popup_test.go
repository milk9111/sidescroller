package system

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestDialoguePopupSystemShowsPopupAtClosestSpeakerTopCenter(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 62, Y: 122, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}

	uiRoot := ecs.CreateEntity(w)
	if err := ecs.Add(w, uiRoot, component.DialogueInputComponent.Kind(), &component.DialogueInput{}); err != nil {
		t.Fatalf("add dialogue input: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.DialoguePopupComponent.Kind(), &component.DialoguePopup{}); err != nil {
		t.Fatalf("add popup component: %v", err)
	}
	if err := ecs.Add(w, popup, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add popup transform: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: true, Image: ebiten.NewImage(16, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	near := ecs.CreateEntity(w)
	if err := ecs.Add(w, near, component.DialogueComponent.Kind(), &component.Dialogue{Range: 100}); err != nil {
		t.Fatalf("add near dialogue: %v", err)
	}
	if err := ecs.Add(w, near, component.TransformComponent.Kind(), &component.Transform{X: 60, Y: 120, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add near transform: %v", err)
	}
	if err := ecs.Add(w, near, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(32, 16), OriginX: 16, OriginY: 8}); err != nil {
		t.Fatalf("add near sprite: %v", err)
	}

	far := ecs.CreateEntity(w)
	if err := ecs.Add(w, far, component.DialogueComponent.Kind(), &component.Dialogue{Range: 100}); err != nil {
		t.Fatalf("add far dialogue: %v", err)
	}
	if err := ecs.Add(w, far, component.TransformComponent.Kind(), &component.Transform{X: 100, Y: 120, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add far transform: %v", err)
	}
	if err := ecs.Add(w, far, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(32, 16), OriginX: 16, OriginY: 8}); err != nil {
		t.Fatalf("add far sprite: %v", err)
	}

	NewDialoguePopupSystem().Update(w)

	popupSprite, ok := ecs.Get(w, popup, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		t.Fatal("expected popup sprite")
	}
	if popupSprite.Disabled {
		t.Fatal("expected popup sprite to be enabled")
	}

	popupTransform, ok := ecs.Get(w, popup, component.TransformComponent.Kind())
	if !ok || popupTransform == nil {
		t.Fatal("expected popup transform")
	}
	if popupTransform.X != 60 || popupTransform.Y != 112 {
		t.Fatalf("expected popup at near speaker top center (60,112), got (%v,%v)", popupTransform.X, popupTransform.Y)
	}
}

func TestDialoguePopupSystemHidesPopupWhenNoSpeakerIsInRange(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}

	uiRoot := ecs.CreateEntity(w)
	if err := ecs.Add(w, uiRoot, component.DialogueInputComponent.Kind(), &component.DialogueInput{}); err != nil {
		t.Fatalf("add dialogue input: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.DialoguePopupComponent.Kind(), &component.DialoguePopup{}); err != nil {
		t.Fatalf("add popup component: %v", err)
	}
	if err := ecs.Add(w, popup, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add popup transform: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false, Image: ebiten.NewImage(16, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	speaker := ecs.CreateEntity(w)
	if err := ecs.Add(w, speaker, component.DialogueComponent.Kind(), &component.Dialogue{Range: 32}); err != nil {
		t.Fatalf("add dialogue: %v", err)
	}
	if err := ecs.Add(w, speaker, component.TransformComponent.Kind(), &component.Transform{X: 200, Y: 200, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add speaker transform: %v", err)
	}
	if err := ecs.Add(w, speaker, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(32, 16), OriginX: 16, OriginY: 8}); err != nil {
		t.Fatalf("add speaker sprite: %v", err)
	}

	NewDialoguePopupSystem().Update(w)

	popupSprite, ok := ecs.Get(w, popup, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		t.Fatal("expected popup sprite")
	}
	if !popupSprite.Disabled {
		t.Fatal("expected popup sprite to be disabled")
	}
}
