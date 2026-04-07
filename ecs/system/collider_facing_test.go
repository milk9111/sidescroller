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

func TestPhysicsSystemFindsPlayerClamberTarget(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	playerTransform := &component.Transform{X: 50, Y: 60, ScaleX: 1, ScaleY: 1}
	playerBody := &component.PhysicsBody{Width: 20, Height: 40}
	playerComp := &component.Player{ClamberInset: 4}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), playerTransform); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), playerBody); err != nil {
		t.Fatalf("add player body: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerComponent.Kind(), playerComp); err != nil {
		t.Fatalf("add player component: %v", err)
	}

	ledge := ecs.CreateEntity(w)
	ledgeTransform := &component.Transform{X: 60, Y: 65, ScaleX: 1, ScaleY: 1}
	ledgeBody := &component.PhysicsBody{Width: 80, Height: 40, AlignTopLeft: true, Static: true}
	if err := ecs.Add(w, ledge, component.TransformComponent.Kind(), ledgeTransform); err != nil {
		t.Fatalf("add ledge transform: %v", err)
	}
	if err := ecs.Add(w, ledge, component.PhysicsBodyComponent.Kind(), ledgeBody); err != nil {
		t.Fatalf("add ledge body: %v", err)
	}

	ps := NewPhysicsSystem()
	targetX, targetY, ok := ps.findPlayerClamberTarget(w, player, playerBody, wallRight)
	if !ok {
		t.Fatal("expected clamber target")
	}
	if math.Abs(targetX-74) > 0.001 || math.Abs(targetY-44.9) > 0.001 {
		t.Fatalf("expected clamber target (74,44.9), got (%v,%v)", targetX, targetY)
	}
}

func TestPhysicsSystemAllowsQuarterBodyClamberTarget(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	playerTransform := &component.Transform{X: 50, Y: 60, ScaleX: 1, ScaleY: 1}
	playerBody := &component.PhysicsBody{Width: 20, Height: 40}
	playerComp := &component.Player{ClamberInset: 4}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), playerTransform); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), playerBody); err != nil {
		t.Fatalf("add player body: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerComponent.Kind(), playerComp); err != nil {
		t.Fatalf("add player component: %v", err)
	}

	ledge := ecs.CreateEntity(w)
	ledgeTransform := &component.Transform{X: 60, Y: 50, ScaleX: 1, ScaleY: 1}
	ledgeBody := &component.PhysicsBody{Width: 80, Height: 40, AlignTopLeft: true, Static: true}
	if err := ecs.Add(w, ledge, component.TransformComponent.Kind(), ledgeTransform); err != nil {
		t.Fatalf("add ledge transform: %v", err)
	}
	if err := ecs.Add(w, ledge, component.PhysicsBodyComponent.Kind(), ledgeBody); err != nil {
		t.Fatalf("add ledge body: %v", err)
	}

	ps := NewPhysicsSystem()
	targetX, targetY, ok := ps.findPlayerClamberTarget(w, player, playerBody, wallRight)
	if !ok {
		t.Fatal("expected clamber target when only a quarter of the body is above the ledge")
	}
	if math.Abs(targetX-74) > 0.001 || math.Abs(targetY-29.9) > 0.001 {
		t.Fatalf("expected quarter-body clamber target (74,29.9), got (%v,%v)", targetX, targetY)
	}
}

func TestPhysicsSystemFindsPlayerClamberTargetOnLeftWall(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	playerTransform := &component.Transform{X: 90, Y: 60, ScaleX: 1, ScaleY: 1}
	playerBody := &component.PhysicsBody{Width: 20, Height: 40}
	playerComp := &component.Player{ClamberInset: 4}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), playerTransform); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), playerBody); err != nil {
		t.Fatalf("add player body: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerComponent.Kind(), playerComp); err != nil {
		t.Fatalf("add player component: %v", err)
	}

	ledge := ecs.CreateEntity(w)
	ledgeTransform := &component.Transform{X: 0, Y: 65, ScaleX: 1, ScaleY: 1}
	ledgeBody := &component.PhysicsBody{Width: 80, Height: 40, AlignTopLeft: true, Static: true}
	if err := ecs.Add(w, ledge, component.TransformComponent.Kind(), ledgeTransform); err != nil {
		t.Fatalf("add ledge transform: %v", err)
	}
	if err := ecs.Add(w, ledge, component.PhysicsBodyComponent.Kind(), ledgeBody); err != nil {
		t.Fatalf("add ledge body: %v", err)
	}

	ps := NewPhysicsSystem()
	targetX, targetY, ok := ps.findBestPlayerClamberTarget(w, player, playerBody, wallNone)
	if !ok {
		t.Fatal("expected clamber target on left wall without relying on a wall contact hint")
	}
	if math.Abs(targetX-66) > 0.001 || math.Abs(targetY-44.9) > 0.001 {
		t.Fatalf("expected left-wall clamber target (66,44.9), got (%v,%v)", targetX, targetY)
	}
}

func TestPhysicsSystemRejectsBlockedClamberTarget(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	playerTransform := &component.Transform{X: 50, Y: 60, ScaleX: 1, ScaleY: 1}
	playerBody := &component.PhysicsBody{Width: 20, Height: 40}
	playerComp := &component.Player{ClamberInset: 4}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), playerTransform); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), playerBody); err != nil {
		t.Fatalf("add player body: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerComponent.Kind(), playerComp); err != nil {
		t.Fatalf("add player component: %v", err)
	}

	ledge := ecs.CreateEntity(w)
	ledgeTransform := &component.Transform{X: 60, Y: 65, ScaleX: 1, ScaleY: 1}
	ledgeBody := &component.PhysicsBody{Width: 80, Height: 40, AlignTopLeft: true, Static: true}
	if err := ecs.Add(w, ledge, component.TransformComponent.Kind(), ledgeTransform); err != nil {
		t.Fatalf("add ledge transform: %v", err)
	}
	if err := ecs.Add(w, ledge, component.PhysicsBodyComponent.Kind(), ledgeBody); err != nil {
		t.Fatalf("add ledge body: %v", err)
	}

	blocker := ecs.CreateEntity(w)
	blockerTransform := &component.Transform{X: 68, Y: 20, ScaleX: 1, ScaleY: 1}
	blockerBody := &component.PhysicsBody{Width: 30, Height: 30, AlignTopLeft: true, Static: true}
	if err := ecs.Add(w, blocker, component.TransformComponent.Kind(), blockerTransform); err != nil {
		t.Fatalf("add blocker transform: %v", err)
	}
	if err := ecs.Add(w, blocker, component.PhysicsBodyComponent.Kind(), blockerBody); err != nil {
		t.Fatalf("add blocker body: %v", err)
	}

	ps := NewPhysicsSystem()
	if _, _, ok := ps.findPlayerClamberTarget(w, player, playerBody, wallRight); ok {
		t.Fatal("expected blocked clamber target to be rejected")
	}
}
