package system

import (
	"math"
	"testing"

	"github.com/jakecoffman/cp"
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

func TestPhysicsBodyCenterUsesTransformForStaticBodies(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	transform := &component.Transform{X: 480, Y: 1728, ScaleX: 1, ScaleY: 1}
	space := cp.NewSpace()
	body := &component.PhysicsBody{
		Body:    space.StaticBody,
		Width:   50,
		Height:  64,
		OffsetX: 16,
		OffsetY: 16,
		Static:  true,
	}

	centerX, centerY, ok := physicsBodyCenter(w, e, transform, body)
	if !ok {
		t.Fatal("expected physics body center")
	}

	if centerX != 496 || centerY != 1744 {
		t.Fatalf("expected static body center (496,1744), got (%v,%v)", centerX, centerY)
	}
}

func TestPhysicsSystemAppliesRotationLockToExistingBody(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	transform := &component.Transform{X: 100, Y: 50, ScaleX: 1, ScaleY: 1}
	body := &component.PhysicsBody{Width: 20, Height: 40, OffsetX: 30, OffsetY: 40, Mass: 1}

	if err := ecs.Add(w, e, component.TransformComponent.Kind(), transform); err != nil {
		t.Fatalf("add transform: %v", err)
	}
	if err := ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), body); err != nil {
		t.Fatalf("add physics body: %v", err)
	}

	ps := NewPhysicsSystem()
	ps.syncEntities(w)

	if body.Body == nil {
		t.Fatal("expected physics body to be created")
	}
	if math.IsInf(body.Body.Moment(), 1) {
		t.Fatal("expected unlocked body to have a finite moment")
	}

	body.LockRotation = true
	body.Body.SetAngularVelocity(3)
	ps.syncEntities(w)

	if !math.IsInf(body.Body.Moment(), 1) {
		t.Fatalf("expected locked body to have infinite moment, got %v", body.Body.Moment())
	}
	if body.Body.AngularVelocity() != 0 {
		t.Fatalf("expected locked body angular velocity to reset to 0, got %v", body.Body.AngularVelocity())
	}
}
