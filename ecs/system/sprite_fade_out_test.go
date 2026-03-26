package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestSpriteFadeOutSystemFadesAndExpires(t *testing.T) {
	w := ecs.NewWorld()
	s := NewSpriteFadeOutSystem()
	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.SpriteFadeOutComponent.Kind(), &component.SpriteFadeOut{Frames: 2, TotalFrames: 2, Alpha: 1})

	s.Update(w)
	fade, ok := ecs.Get(w, e, component.SpriteFadeOutComponent.Kind())
	if !ok || fade == nil {
		t.Fatal("expected sprite fade out component to remain active after first update")
	}
	if fade.Frames != 1 {
		t.Fatalf("expected one frame remaining, got %d", fade.Frames)
	}
	if fade.Alpha != 0.5 {
		t.Fatalf("expected alpha to be 0.5 after first update, got %v", fade.Alpha)
	}

	s.Update(w)
	fade, ok = ecs.Get(w, e, component.SpriteFadeOutComponent.Kind())
	if !ok || fade == nil {
		t.Fatal("expected sprite fade out component to remain active at zero alpha before expiry")
	}
	if fade.Frames != 0 {
		t.Fatalf("expected zero frames remaining, got %d", fade.Frames)
	}
	if fade.Alpha != 0 {
		t.Fatalf("expected alpha to reach 0, got %v", fade.Alpha)
	}

	s.Update(w)
	if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected sprite fade out component to be removed after duration elapses")
	}
}

func TestSpriteFadeOutSystemMarksStaticTileBatchDirty(t *testing.T) {
	w := ecs.NewWorld()
	s := NewSpriteFadeOutSystem()
	bounds := ecs.CreateEntity(w)
	_ = ecs.Add(w, bounds, component.LevelGridComponent.Kind(), &component.LevelGrid{})
	_ = ecs.Add(w, bounds, component.StaticTileBatchStateComponent.Kind(), &component.StaticTileBatchState{Dirty: false})

	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.StaticTileComponent.Kind(), &component.StaticTile{})
	_ = ecs.Add(w, e, component.SpriteFadeOutComponent.Kind(), &component.SpriteFadeOut{Frames: 1, TotalFrames: 1, Alpha: 1})

	s.Update(w)
	state, ok := ecs.Get(w, bounds, component.StaticTileBatchStateComponent.Kind())
	if !ok || state == nil || !state.Dirty {
		t.Fatal("expected sprite fade out update to mark the static tile batch dirty")
	}
}
