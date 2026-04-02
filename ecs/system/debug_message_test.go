package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
)

func TestDebugMessageSystemHidesExpiredMessage(t *testing.T) {
	w := ecs.NewWorld()
	if _, err := levelentity.NewDebugMessage(w); err != nil {
		t.Fatalf("new debug message: %v", err)
	}
	if err := levelentity.ShowTimedDebugMessage(w, 240, 40, "test message", 2); err != nil {
		t.Fatalf("show timed debug message: %v", err)
	}

	ent, ok := ecs.First(w, component.DebugMessageComponent.Kind())
	if !ok {
		t.Fatal("expected debug message entity")
	}

	system := NewDebugMessageSystem()
	system.Update(w)

	debugMessage, _ := ecs.Get(w, ent, component.DebugMessageComponent.Kind())
	if debugMessage == nil || debugMessage.RemainingFrames != 1 {
		t.Fatalf("expected one frame remaining, got %+v", debugMessage)
	}

	system.Update(w)

	debugMessage, _ = ecs.Get(w, ent, component.DebugMessageComponent.Kind())
	if debugMessage == nil || debugMessage.Message != "" || debugMessage.RemainingFrames != 0 {
		t.Fatalf("expected cleared message state, got %+v", debugMessage)
	}

	sprite, _ := ecs.Get(w, ent, component.SpriteComponent.Kind())
	if sprite == nil || !sprite.Disabled || sprite.Image != nil {
		t.Fatalf("expected hidden sprite after expiry, got %+v", sprite)
	}
}
