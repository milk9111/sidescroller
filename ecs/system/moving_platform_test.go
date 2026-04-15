package system

import (
	"math"
	"testing"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestMovingPlatformSystemContinuousModeMovesAndReverses(t *testing.T) {
	w := ecs.NewWorld()
	platform := ecs.CreateEntity(w)
	if err := ecs.Add(w, platform, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transform: %v", err)
	}
	if err := ecs.Add(w, platform, component.MovingPlatformComponent.Kind(), &component.MovingPlatform{Mode: component.MovingPlatformModeContinuous, Speed: 32, DestX: 96}); err != nil {
		t.Fatalf("add moving platform: %v", err)
	}

	system := NewMovingPlatformSystem()
	for range 3 {
		system.Update(w)
	}

	transform, _ := ecs.Get(w, platform, component.TransformComponent.Kind())
	if transform.X != 96 || transform.Y != 0 {
		t.Fatalf("expected platform to reach destination at (96,0), got (%.1f,%.1f)", transform.X, transform.Y)
	}

	system.Update(w)
	if transform.X != 64 {
		t.Fatalf("expected platform to reverse back toward start, got x=%.1f", transform.X)
	}
}

func TestMovingPlatformSystemTouchModeStartsOnInitialTouch(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	platform := ecs.CreateEntity(w)

	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerCollisionComponent.Kind(), &component.PlayerCollision{Grounded: true, GroundEntity: uint64(platform)}); err != nil {
		t.Fatalf("add player collision: %v", err)
	}
	if err := ecs.Add(w, platform, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add platform transform: %v", err)
	}
	if err := ecs.Add(w, platform, component.MovingPlatformComponent.Kind(), &component.MovingPlatform{Mode: component.MovingPlatformModeTouch, Speed: 32, DestX: 96}); err != nil {
		t.Fatalf("add moving platform: %v", err)
	}

	system := NewMovingPlatformSystem()
	system.Update(w)
	platformTransform, _ := ecs.Get(w, platform, component.TransformComponent.Kind())
	playerTransform, _ := ecs.Get(w, player, component.TransformComponent.Kind())
	if platformTransform.X != 32 {
		t.Fatalf("expected touch mode platform to start moving as soon as the player touches it, got x=%.1f", platformTransform.X)
	}
	if playerTransform.X != 32 {
		t.Fatalf("expected grounded rider to be carried immediately with the platform, got x=%.1f", playerTransform.X)
	}

	playerCollision, _ := ecs.Get(w, player, component.PlayerCollisionComponent.Kind())

	playerCollision.Grounded = true
	playerCollision.GroundEntity = uint64(platform)
	system.Update(w)
	if playerTransform.X != 64 {
		t.Fatalf("expected grounded rider to continue being carried with the platform, got x=%.1f", playerTransform.X)
	}
}

func TestMovingPlatformSystemTouchModeDoesNotRestartUntilPlayerReenters(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	platform := ecs.CreateEntity(w)

	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerCollisionComponent.Kind(), &component.PlayerCollision{Grounded: true, GroundEntity: uint64(platform)}); err != nil {
		t.Fatalf("add player collision: %v", err)
	}
	if err := ecs.Add(w, platform, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add platform transform: %v", err)
	}
	if err := ecs.Add(w, platform, component.MovingPlatformComponent.Kind(), &component.MovingPlatform{Mode: component.MovingPlatformModeTouch, Speed: 32, DestX: 32}); err != nil {
		t.Fatalf("add moving platform: %v", err)
	}

	system := NewMovingPlatformSystem()
	system.Update(w)
	platformTransform, _ := ecs.Get(w, platform, component.TransformComponent.Kind())
	if platformTransform.X != 32 {
		t.Fatalf("expected touch mode platform to reach its destination, got x=%.1f", platformTransform.X)
	}

	system.Update(w)
	if platformTransform.X != 32 {
		t.Fatalf("expected touch mode platform to remain stopped at destination while player stays on it, got x=%.1f", platformTransform.X)
	}

	playerCollision, _ := ecs.Get(w, player, component.PlayerCollisionComponent.Kind())
	playerCollision.Grounded = false
	playerCollision.GroundEntity = 0
	system.Update(w)
	if platformTransform.X != 32 {
		t.Fatalf("expected touch mode platform to remain stopped after player leaves, got x=%.1f", platformTransform.X)
	}

	playerCollision.Grounded = true
	playerCollision.GroundEntity = uint64(platform)
	system.Update(w)
	if platformTransform.X != 0 {
		t.Fatalf("expected touch mode platform to restart only after player reenters, got x=%.1f", platformTransform.X)
	}
}

func TestMovingPlatformSystemMovesDynamicPhysicsBodyAtConfiguredSpeed(t *testing.T) {
	w := ecs.NewWorld()
	platform := ecs.CreateEntity(w)

	if err := ecs.Add(w, platform, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add platform transform: %v", err)
	}
	if err := ecs.Add(w, platform, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 90, Height: 28, Mass: 1000, LockRotation: true, AlignTopLeft: true, OffsetX: 3, OffsetY: 2}); err != nil {
		t.Fatalf("add physics body: %v", err)
	}
	if err := ecs.Add(w, platform, component.GravityScaleComponent.Kind(), &component.GravityScale{Scale: 1}); err != nil {
		t.Fatalf("add gravity scale: %v", err)
	}
	if err := ecs.Add(w, platform, component.MovingPlatformComponent.Kind(), &component.MovingPlatform{Mode: component.MovingPlatformModeContinuous, Speed: 10.5, DestY: -704}); err != nil {
		t.Fatalf("add moving platform: %v", err)
	}

	movingPlatformSystem := NewMovingPlatformSystem()
	physicsSystem := NewPhysicsSystem()
	physicsSystem.Update(w)
	movingPlatformSystem.Update(w)
	physicsSystem.Update(w)

	transform, _ := ecs.Get(w, platform, component.TransformComponent.Kind())
	if math.Abs(transform.Y-(-10.5)) > 1.0 {
		t.Fatalf("expected dynamic physics body platform to move close to -10.5 units in one update, got y=%.2f", transform.Y)
	}

	body, _ := ecs.Get(w, platform, component.PhysicsBodyComponent.Kind())
	if body == nil || body.Body == nil {
		t.Fatal("expected dynamic physics body to be created")
	}
	velocity := body.Body.Velocity()
	if math.Abs(velocity.Y-(-10.5)) > 0.0001 {
		t.Fatalf("expected dynamic physics body velocity -10.5, got %.2f", velocity.Y)
	}
}

func TestMovingPlatformSystemDynamicBodySnapsToDestinationWithoutRestarting(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	platform := ecs.CreateEntity(w)

	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerCollisionComponent.Kind(), &component.PlayerCollision{Grounded: true, GroundEntity: uint64(platform)}); err != nil {
		t.Fatalf("add player collision: %v", err)
	}
	if err := ecs.Add(w, platform, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add platform transform: %v", err)
	}
	if err := ecs.Add(w, platform, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 90, Height: 28, Mass: 1000, LockRotation: true, AlignTopLeft: true, OffsetX: 3, OffsetY: 2}); err != nil {
		t.Fatalf("add platform physics body: %v", err)
	}
	if err := ecs.Add(w, platform, component.GravityScaleComponent.Kind(), &component.GravityScale{Scale: 1}); err != nil {
		t.Fatalf("add gravity scale: %v", err)
	}
	if err := ecs.Add(w, platform, component.MovingPlatformComponent.Kind(), &component.MovingPlatform{Mode: component.MovingPlatformModeTouch, Speed: 32, DestX: 32}); err != nil {
		t.Fatalf("add moving platform: %v", err)
	}

	movingPlatformSystem := NewMovingPlatformSystem()
	physicsSystem := NewPhysicsSystem()

	physicsSystem.Update(w)
	playerCollision, _ := ecs.Get(w, player, component.PlayerCollisionComponent.Kind())
	playerCollision.Grounded = true
	playerCollision.GroundEntity = uint64(platform)
	movingPlatformSystem.Update(w)
	physicsSystem.Update(w)

	transform, _ := ecs.Get(w, platform, component.TransformComponent.Kind())
	if math.Abs(transform.X-32) > 0.0001 {
		t.Fatalf("expected dynamic touch platform to reach exact destination x=32, got %.3f", transform.X)
	}

	playerCollision.Grounded = true
	playerCollision.GroundEntity = uint64(platform)
	movingPlatformSystem.Update(w)
	physicsSystem.Update(w)

	if math.Abs(transform.X-32) > 0.0001 {
		t.Fatalf("expected dynamic touch platform to remain at destination while rider stays on it, got %.3f", transform.X)
	}

	body, _ := ecs.Get(w, platform, component.PhysicsBodyComponent.Kind())
	velocity := body.Body.Velocity()
	if math.Abs(velocity.X) > 0.0001 || math.Abs(velocity.Y) > 0.0001 {
		t.Fatalf("expected stopped platform body velocity to remain zero at destination, got (%.3f, %.3f)", velocity.X, velocity.Y)
	}
}

func TestMovingPlatformSystemDoesNotAccumulateGroundVelocity(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	platform := ecs.CreateEntity(w)

	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 40, Mass: 1}); err != nil {
		t.Fatalf("add player physics body: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerCollisionComponent.Kind(), &component.PlayerCollision{Grounded: true, GroundEntity: uint64(platform)}); err != nil {
		t.Fatalf("add player collision: %v", err)
	}
	if err := ecs.Add(w, platform, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add platform transform: %v", err)
	}
	if err := ecs.Add(w, platform, component.MovingPlatformComponent.Kind(), &component.MovingPlatform{Mode: component.MovingPlatformModeContinuous, Speed: 32, DestX: 96}); err != nil {
		t.Fatalf("add moving platform: %v", err)
	}

	playerBody, _ := ecs.Get(w, player, component.PhysicsBodyComponent.Kind())
	playerBody.Body = cp.NewBody(playerBody.Mass, 1)

	system := NewMovingPlatformSystem()
	system.Update(w)

	velocity := playerBody.Body.Velocity()
	if velocity.X != 32 {
		t.Fatalf("expected grounded player to inherit platform velocity 32 on first frame, got %.1f", velocity.X)
	}

	system.Update(w)
	velocity = playerBody.Body.Velocity()
	if velocity.X != 32 {
		t.Fatalf("expected grounded player to keep inherited platform velocity at 32 instead of accumulating, got %.1f", velocity.X)
	}

	playerCollision, _ := ecs.Get(w, player, component.PlayerCollisionComponent.Kind())
	if playerCollision.GroundVelocityX != 32 || playerCollision.GroundVelocityY != 0 {
		t.Fatalf("expected stored ground velocity (32,0), got (%.1f,%.1f)", playerCollision.GroundVelocityX, playerCollision.GroundVelocityY)
	}

	playerCollision.Grounded = false
	playerCollision.GroundEntity = 0
	system.Update(w)
	velocity = playerBody.Body.Velocity()
	if velocity.X != 0 {
		t.Fatalf("expected carried ground velocity to be removed after leaving platform, got %.1f", velocity.X)
	}
	if playerCollision.GroundVelocityX != 0 || playerCollision.GroundVelocityY != 0 {
		t.Fatalf("expected stored ground velocity to clear after leaving platform, got (%.1f,%.1f)", playerCollision.GroundVelocityX, playerCollision.GroundVelocityY)
	}
}
