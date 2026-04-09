package system

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestInsideTransitionShowsPopupAndRequiresUpPress(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 8, Y: 8, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, AlignTopLeft: true}); err != nil {
		t.Fatalf("add player body: %v", err)
	}
	if err := ecs.Add(w, player, component.SpriteComponent.Kind(), &component.Sprite{}); err != nil {
		t.Fatalf("add player sprite: %v", err)
	}

	transition := ecs.CreateEntity(w)
	if err := ecs.Add(w, transition, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transition transform: %v", err)
	}
	if err := ecs.Add(w, transition, component.TransitionComponent.Kind(), &component.Transition{
		ID:          "inside_door",
		TargetLevel: "interior.json",
		LinkedID:    "door_exit",
		EnterDir:    component.TransitionDirDown,
		Type:        component.TransitionTypeInside,
		Bounds:      component.AABB{W: 64, H: 64},
	}); err != nil {
		t.Fatalf("add transition: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.TransitionPopupComponent.Kind(), &component.TransitionPopup{Base: ebiten.NewImage(16, 8)}); err != nil {
		t.Fatalf("add transition popup: %v", err)
	}
	if err := ecs.Add(w, popup, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add popup transform: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: true, Image: ebiten.NewImage(16, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	input := ecs.CreateEntity(w)
	if err := ecs.Add(w, input, component.TransitionInputComponent.Kind(), &component.TransitionInput{}); err != nil {
		t.Fatalf("add transition input: %v", err)
	}

	NewTransitionPopupSystem().Update(w)

	popupComp, ok := ecs.Get(w, popup, component.TransitionPopupComponent.Kind())
	if !ok || popupComp == nil {
		t.Fatal("expected transition popup component")
	}
	if popupComp.TargetTransitionEntity != uint64(transition) {
		t.Fatalf("expected popup to target inside transition, got %d", popupComp.TargetTransitionEntity)
	}
	popupSprite, ok := ecs.Get(w, popup, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		t.Fatal("expected popup sprite")
	}
	if popupSprite.Disabled {
		t.Fatal("expected popup sprite to be enabled")
	}

	NewTransitionSystem().Update(w)
	if _, ok := ecs.First(w, component.TransitionRuntimeComponent.Kind()); ok {
		t.Fatal("expected inside transition to stay idle until up is pressed")
	}

	transitionInput, _ := ecs.Get(w, input, component.TransitionInputComponent.Kind())
	transitionInput.UpPressed = true
	if err := ecs.Add(w, input, component.TransitionInputComponent.Kind(), transitionInput); err != nil {
		t.Fatalf("update transition input: %v", err)
	}

	NewTransitionSystem().Update(w)
	runtimeEntity, ok := ecs.First(w, component.TransitionRuntimeComponent.Kind())
	if !ok {
		t.Fatal("expected inside transition to create a runtime after up is pressed")
	}
	runtime, ok := ecs.Get(w, runtimeEntity, component.TransitionRuntimeComponent.Kind())
	if !ok || runtime == nil {
		t.Fatal("expected transition runtime")
	}
	if runtime.Req.TargetLevel != "interior.json" || runtime.Req.SpawnTransitionID != "door_exit" {
		t.Fatalf("unexpected level change request: %+v", runtime.Req)
	}
}

func TestTransitionAABBPersistsSubTileTriggerBounds(t *testing.T) {
	w := ecs.NewWorld()
	transition := ecs.CreateEntity(w)
	if err := ecs.Add(w, transition, component.TransformComponent.Kind(), &component.Transform{X: 100, Y: 200, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transition transform: %v", err)
	}

	box := transitionAABB(w, transition, &component.Transition{
		Bounds: component.AABB{X: 10, Y: 0, W: 22, H: 32},
	})

	if box.x != 110 || box.y != 200 || box.w != 22 || box.h != 32 {
		t.Fatalf("expected sub-tile trigger bounds to be preserved, got %+v", box)
	}
}
