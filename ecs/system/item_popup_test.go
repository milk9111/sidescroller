package system

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestItemPopupSystemShowsPopupAtClosestItemTopCenter(t *testing.T) {
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
		t.Fatalf("add interaction input: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.ItemPopupComponent.Kind(), &component.ItemPopup{}); err != nil {
		t.Fatalf("add popup component: %v", err)
	}
	if err := ecs.Add(w, popup, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add popup transform: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: true, Image: ebiten.NewImage(16, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	near := ecs.CreateEntity(w)
	if err := ecs.Add(w, near, component.ItemComponent.Kind(), &component.Item{Range: 100}); err != nil {
		t.Fatalf("add near item: %v", err)
	}
	if err := ecs.Add(w, near, component.TransformComponent.Kind(), &component.Transform{X: 60, Y: 120, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add near transform: %v", err)
	}
	if err := ecs.Add(w, near, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(32, 16), OriginX: 16, OriginY: 8}); err != nil {
		t.Fatalf("add near sprite: %v", err)
	}

	far := ecs.CreateEntity(w)
	if err := ecs.Add(w, far, component.ItemComponent.Kind(), &component.Item{Range: 100}); err != nil {
		t.Fatalf("add far item: %v", err)
	}
	if err := ecs.Add(w, far, component.TransformComponent.Kind(), &component.Transform{X: 100, Y: 120, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add far transform: %v", err)
	}
	if err := ecs.Add(w, far, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(32, 16), OriginX: 16, OriginY: 8}); err != nil {
		t.Fatalf("add far sprite: %v", err)
	}

	NewItemPopupSystem().Update(w)

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
	if popupTransform.X != 60 || popupTransform.Y != 106 {
		t.Fatalf("expected popup above near item top center (60,106), got (%v,%v)", popupTransform.X, popupTransform.Y)
	}
}

func TestItemPopupSystemHidesPopupWhenNoItemIsInRange(t *testing.T) {
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
		t.Fatalf("add interaction input: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.ItemPopupComponent.Kind(), &component.ItemPopup{}); err != nil {
		t.Fatalf("add popup component: %v", err)
	}
	if err := ecs.Add(w, popup, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add popup transform: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false, Image: ebiten.NewImage(16, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	item := ecs.CreateEntity(w)
	if err := ecs.Add(w, item, component.ItemComponent.Kind(), &component.Item{Range: 32}); err != nil {
		t.Fatalf("add item: %v", err)
	}
	if err := ecs.Add(w, item, component.TransformComponent.Kind(), &component.Transform{X: 200, Y: 200, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add item transform: %v", err)
	}
	if err := ecs.Add(w, item, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(32, 16), OriginX: 16, OriginY: 8}); err != nil {
		t.Fatalf("add item sprite: %v", err)
	}

	NewItemPopupSystem().Update(w)

	popupSprite, ok := ecs.Get(w, popup, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		t.Fatal("expected popup sprite")
	}
	if !popupSprite.Disabled {
		t.Fatal("expected popup sprite to be disabled")
	}
}
