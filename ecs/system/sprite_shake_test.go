package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestSpriteShakeSystemRandomizesAndExpires(t *testing.T) {
	w := ecs.NewWorld()
	s := NewSpriteShakeSystem()
	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.SpriteShakeComponent.Kind(), &component.SpriteShake{Frames: 2, Intensity: 3})

	s.Update(w)
	shake, ok := ecs.Get(w, e, component.SpriteShakeComponent.Kind())
	if !ok || shake == nil {
		t.Fatal("expected sprite shake component to remain active after first update")
	}
	if shake.Frames != 1 {
		t.Fatalf("expected one frame remaining, got %d", shake.Frames)
	}
	if shake.OffsetX < -3 || shake.OffsetX > 3 {
		t.Fatalf("expected x offset within intensity bounds, got %v", shake.OffsetX)
	}
	if shake.OffsetY < -3 || shake.OffsetY > 3 {
		t.Fatalf("expected y offset within intensity bounds, got %v", shake.OffsetY)
	}

	s.Update(w)
	if ecs.Has(w, e, component.SpriteShakeComponent.Kind()) {
		t.Fatal("expected sprite shake component to be removed after duration elapses")
	}
}

func TestSpriteShakeSystemMarksStaticTileBatchDirty(t *testing.T) {
	w := ecs.NewWorld()
	s := NewSpriteShakeSystem()
	bounds := ecs.CreateEntity(w)
	_ = ecs.Add(w, bounds, component.LevelGridComponent.Kind(), &component.LevelGrid{})
	_ = ecs.Add(w, bounds, component.StaticTileBatchStateComponent.Kind(), &component.StaticTileBatchState{Dirty: false})

	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.StaticTileComponent.Kind(), &component.StaticTile{})
	_ = ecs.Add(w, e, component.SpriteShakeComponent.Kind(), &component.SpriteShake{Frames: 1, Intensity: 2})

	s.Update(w)
	state, ok := ecs.Get(w, bounds, component.StaticTileBatchStateComponent.Kind())
	if !ok || state == nil || !state.Dirty {
		t.Fatal("expected sprite shake update to mark the static tile batch dirty")
	}
}
