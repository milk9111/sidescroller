package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestPhysicsBodyBoundsUsesCenteredOffsets(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	transform := &component.Transform{X: 100, Y: 50, ScaleX: 1, ScaleY: 1}
	body := &component.PhysicsBody{Width: 20, Height: 40, OffsetX: 30, OffsetY: 40}

	minX, minY, maxX, maxY, ok := physicsBodyBounds(w, e, transform, body)
	if !ok {
		t.Fatal("expected physics body bounds")
	}
	if minX != 120 || minY != 70 || maxX != 140 || maxY != 110 {
		t.Fatalf("expected centered bounds (120,70)-(140,110), got (%v,%v)-(%v,%v)", minX, minY, maxX, maxY)
	}

	centerX := bodyCenterX(w, e, transform, body)
	centerY := bodyCenterY(transform, body)
	if centerX != 130 || centerY != 90 {
		t.Fatalf("expected centered body position (130,90), got (%v,%v)", centerX, centerY)
	}
}

func TestPhysicsBodyBoundsLegacyTopLeftCompatibility(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	transform := &component.Transform{X: 100, Y: 50, ScaleX: 1, ScaleY: 1}
	body := &component.PhysicsBody{Width: 20, Height: 40, OffsetX: 20, OffsetY: 20, AlignTopLeft: true}

	minX, minY, maxX, maxY, ok := physicsBodyBounds(w, e, transform, body)
	if !ok {
		t.Fatal("expected physics body bounds")
	}
	if minX != 120 || minY != 70 || maxX != 140 || maxY != 110 {
		t.Fatalf("expected legacy bounds to match previous world rect, got (%v,%v)-(%v,%v)", minX, minY, maxX, maxY)
	}

	centerX := bodyCenterX(w, e, transform, body)
	centerY := bodyCenterY(transform, body)
	if centerX != 130 || centerY != 90 {
		t.Fatalf("expected legacy body center (130,90), got (%v,%v)", centerX, centerY)
	}
}
