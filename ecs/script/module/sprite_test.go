package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestSpriteModuleAddFadeOutAddsComponent(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	mod := SpriteModule().Build(w, nil, e, e)

	if _, err := mod["add_fade_out"].(*tengo.UserFunction).Value(&tengo.Int{Value: 12}); err != nil {
		t.Fatalf("add_fade_out returned error: %v", err)
	}

	fade, ok := ecs.Get(w, e, component.SpriteFadeOutComponent.Kind())
	if !ok || fade == nil {
		t.Fatal("expected sprite fade out component to be added")
	}
	if fade.Frames != 12 {
		t.Fatalf("expected 12 frames, got %d", fade.Frames)
	}
	if fade.TotalFrames != 12 {
		t.Fatalf("expected total frames to be 12, got %d", fade.TotalFrames)
	}
	if fade.Alpha != 1 {
		t.Fatalf("expected fade alpha to start at 1, got %v", fade.Alpha)
	}
}
