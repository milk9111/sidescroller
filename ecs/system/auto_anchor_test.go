package system

import (
	"math"
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestClosestAutoAnchorTargetChoosesNearestReachableSurface(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)

	addStaticAnchorSurface(t, w, 40, 0, 20, 20)
	addStaticAnchorSurface(t, w, 80, 0, 20, 20)

	hitX, hitY, ok := closestAutoAnchorTarget(w, player, 0, 0, 0, 100, 0, 0)
	if !ok {
		t.Fatal("expected an auto-anchor target")
	}
	if !nearlyEqual(hitX, 30) || !nearlyEqual(hitY, 0) {
		t.Fatalf("expected nearest surface hit at (30, 0), got (%.3f, %.3f)", hitX, hitY)
	}
}

func TestClosestAutoAnchorTargetSkipsHazardBlockedSurface(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)

	addStaticAnchorSurface(t, w, 60, 0, 20, 20)
	addStaticAnchorSurface(t, w, 0, 80, 20, 20)
	addHazardBlocker(t, w, 20, 0, 12, 12)

	hitX, hitY, ok := closestAutoAnchorTarget(w, player, 0, 0, 0, 100, 0, 0)
	if !ok {
		t.Fatal("expected an auto-anchor target")
	}
	if !nearlyEqual(hitX, 0) || !nearlyEqual(hitY, 70) {
		t.Fatalf("expected hazard-blocked target to be skipped in favor of (0, 70), got (%.3f, %.3f)", hitX, hitY)
	}
}

func TestClosestAutoAnchorTargetHonorsMaxDistance(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)

	addStaticAnchorSurface(t, w, 80, 0, 20, 20)

	if _, _, ok := closestAutoAnchorTarget(w, player, 0, 0, 0, 25, 0, 0); ok {
		t.Fatal("expected no auto-anchor target beyond max distance")
	}
}

func TestClosestAutoAnchorTargetHonorsMinDistance(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)

	addStaticAnchorSurface(t, w, 20, 0, 20, 20)
	addStaticAnchorSurface(t, w, 0, 60, 20, 20)

	hitX, hitY, ok := closestAutoAnchorTarget(w, player, 0, 0, 25, 100, 0, 0)
	if !ok {
		t.Fatal("expected an auto-anchor target beyond minimum length")
	}
	if !nearlyEqual(hitX, 0) || !nearlyEqual(hitY, 50) {
		t.Fatalf("expected too-close surface to be skipped in favor of (0, 50), got (%.3f, %.3f)", hitX, hitY)
	}
}

func TestClosestAutoAnchorTargetUsesPlayerBodyOriginForMinDistance(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 20}); err != nil {
		t.Fatalf("add player body: %v", err)
	}

	addStaticAnchorSurface(t, w, 0, 20, 20, 20)
	addStaticAnchorSurface(t, w, 60, 0, 20, 20)

	hitX, hitY, ok := closestAutoAnchorTarget(w, player, 0, -30, 25, 100, 0, 0)
	if !ok {
		t.Fatal("expected an auto-anchor target beyond minimum length from the body origin")
	}
	if !nearlyEqual(hitX, 50) || !nearlyEqual(hitY, 0) {
		t.Fatalf("expected body-origin min distance to reject the near ground point and choose (50, 0), got (%.3f, %.3f)", hitX, hitY)
	}
}

func TestClosestAutoAnchorTargetPrefersMovementDirection(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)

	addStaticAnchorSurface(t, w, -30, 0, 20, 20)
	addStaticAnchorSurface(t, w, 60, 0, 20, 20)

	hitX, hitY, ok := closestAutoAnchorTarget(w, player, 0, 0, 0, 100, 1, 0)
	if !ok {
		t.Fatal("expected an auto-anchor target in the movement direction")
	}
	if !nearlyEqual(hitX, 50) || !nearlyEqual(hitY, 0) {
		t.Fatalf("expected rightward movement to prefer (50, 0), got (%.3f, %.3f)", hitX, hitY)
	}
}

func TestClosestAutoAnchorTargetNarrowsForwardCone(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)

	addStaticAnchorSurface(t, w, 30, 40, 20, 20)
	addStaticAnchorSurface(t, w, 60, 0, 20, 20)

	hitX, hitY, ok := closestAutoAnchorTarget(w, player, 0, 0, 0, 100, 1, 0)
	if !ok {
		t.Fatal("expected an auto-anchor target in the narrowed forward cone")
	}
	if !nearlyEqual(hitX, 50) || !nearlyEqual(hitY, 0) {
		t.Fatalf("expected narrower forward cone to prefer the more extreme forward target at (50, 0), got (%.3f, %.3f)", hitX, hitY)
	}
}

func TestClosestAutoAnchorTargetFallsBackToNearestWhenNoForwardTarget(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)

	addStaticAnchorSurface(t, w, -30, 0, 20, 20)
	addStaticAnchorSurface(t, w, -60, 0, 20, 20)

	hitX, hitY, ok := closestAutoAnchorTarget(w, player, 0, 0, 0, 100, 1, 0)
	if !ok {
		t.Fatal("expected an auto-anchor fallback target")
	}
	if !nearlyEqual(hitX, -20) || !nearlyEqual(hitY, 0) {
		t.Fatalf("expected nearest fallback target at (-20, 0), got (%.3f, %.3f)", hitX, hitY)
	}
}

func addStaticAnchorSurface(t *testing.T, w *ecs.World, x, y, width, height float64) ecs.Entity {
	t.Helper()
	e := ecs.CreateEntity(w)
	if err := ecs.Add(w, e, component.TransformComponent.Kind(), &component.Transform{X: x, Y: y, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transform: %v", err)
	}
	if err := ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Static: true, Width: width, Height: height}); err != nil {
		t.Fatalf("add physics body: %v", err)
	}
	return e
}

func addHazardBlocker(t *testing.T, w *ecs.World, x, y, width, height float64) ecs.Entity {
	t.Helper()
	e := ecs.CreateEntity(w)
	if err := ecs.Add(w, e, component.TransformComponent.Kind(), &component.Transform{X: x, Y: y, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transform: %v", err)
	}
	if err := ecs.Add(w, e, component.SpikeTagComponent.Kind(), &component.SpikeTag{}); err != nil {
		t.Fatalf("add spike tag: %v", err)
	}
	if err := ecs.Add(w, e, component.HazardComponent.Kind(), &component.Hazard{Width: width, Height: height}); err != nil {
		t.Fatalf("add hazard: %v", err)
	}
	return e
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}
