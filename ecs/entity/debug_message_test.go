package entity

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestNewDebugMessageStartsHiddenAndPersistent(t *testing.T) {
	w := ecs.NewWorld()

	ent, err := NewDebugMessage(w)
	if err != nil {
		t.Fatalf("new debug message: %v", err)
	}

	persistent, ok := ecs.Get(w, ent, component.PersistentComponent.Kind())
	if !ok || persistent == nil {
		t.Fatal("expected persistent component")
	}
	if persistent.ID != "debug_message" || !persistent.KeepOnLevelChange {
		t.Fatalf("unexpected persistent settings: %+v", persistent)
	}

	debugMessage, ok := ecs.Get(w, ent, component.DebugMessageComponent.Kind())
	if !ok || debugMessage == nil {
		t.Fatal("expected debug message component")
	}
	if debugMessage.Width != DebugMessageDefaultWidth || debugMessage.Height != DebugMessageDefaultHeight {
		t.Fatalf("unexpected default dimensions: %+v", debugMessage)
	}

	sprite, ok := ecs.Get(w, ent, component.SpriteComponent.Kind())
	if !ok || sprite == nil {
		t.Fatal("expected sprite component")
	}
	if !sprite.Disabled {
		t.Fatal("expected debug message sprite to start hidden")
	}

	transform, ok := ecs.Get(w, ent, component.TransformComponent.Kind())
	if !ok || transform == nil {
		t.Fatal("expected transform component")
	}
	if transform.Y != DebugMessageTopY {
		t.Fatalf("expected transform y=%v, got %v", DebugMessageTopY, transform.Y)
	}
}
