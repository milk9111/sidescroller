package entity

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestBuildEntityCentersAnimatedSpriteOriginWhenRequested(t *testing.T) {
	w := ecs.NewWorld()
	e, err := BuildEntity(w, "electric_field.yaml")
	if err != nil {
		t.Fatalf("build entity: %v", err)
	}

	sprite, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
	if !ok || sprite == nil {
		t.Fatal("expected sprite component")
	}
	if sprite.OriginX != 32 || sprite.OriginY != 32 {
		t.Fatalf("expected animated sprite origin to be centered at (32,32), got (%v,%v)", sprite.OriginX, sprite.OriginY)
	}
}

func TestBuildEntityKeepsExplicitAnimatedSpriteOrigin(t *testing.T) {
	w := ecs.NewWorld()
	e, err := BuildEntityWithOverrides(w, "electric_field.yaml", map[string]any{
		"sprite": map[string]any{
			"origin_x":              10.0,
			"origin_y":              14.0,
			"center_origin_if_zero": true,
		},
	})
	if err != nil {
		t.Fatalf("build entity with overrides: %v", err)
	}

	sprite, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
	if !ok || sprite == nil {
		t.Fatal("expected sprite component")
	}
	if sprite.OriginX != 10 || sprite.OriginY != 14 {
		t.Fatalf("expected explicit sprite origin to be preserved, got (%v,%v)", sprite.OriginX, sprite.OriginY)
	}
}