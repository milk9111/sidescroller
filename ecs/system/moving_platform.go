package system

import (
	"math"
	"strings"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const movingPlatformEpsilon = 1e-6

type MovingPlatformSystem struct{}

func NewMovingPlatformSystem() *MovingPlatformSystem { return &MovingPlatformSystem{} }

func (s *MovingPlatformSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	playerEntity, hasPlayer := ecs.First(w, component.PlayerTagComponent.Kind())
	var playerCollision *component.PlayerCollision
	var playerTransform *component.Transform
	var playerPhysicsBody *component.PhysicsBody
	if hasPlayer {
		playerCollision, _ = ecs.Get(w, playerEntity, component.PlayerCollisionComponent.Kind())
		playerTransform, _ = ecs.Get(w, playerEntity, component.TransformComponent.Kind())
		playerPhysicsBody, _ = ecs.Get(w, playerEntity, component.PhysicsBodyComponent.Kind())
	}

	if playerCollision != nil && playerPhysicsBody != nil && playerPhysicsBody.Body != nil {
		previousGroundVelocityX := playerCollision.GroundVelocityX
		previousGroundVelocityY := playerCollision.GroundVelocityY
		if math.Abs(previousGroundVelocityX) > movingPlatformEpsilon || math.Abs(previousGroundVelocityY) > movingPlatformEpsilon {
			velocity := playerPhysicsBody.Body.Velocity()
			playerPhysicsBody.Body.SetVelocity(velocity.X-previousGroundVelocityX, velocity.Y-previousGroundVelocityY)
		}
		playerCollision.GroundVelocityX = 0
		playerCollision.GroundVelocityY = 0
	}

	ecs.ForEach2(w, component.MovingPlatformComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, platform *component.MovingPlatform, transform *component.Transform) {
		if platform == nil || transform == nil {
			return
		}

		mode := normalizeMovingPlatformMode(platform.Mode)
		initializeMovingPlatform(platform, transform, mode)
		currentX, currentY := movingPlatformPosition(platform)
		if movingPlatformUsesBody(w, e) {
			transform.X = currentX
			transform.Y = currentY
		}

		currentTouching := playerCollision != nil && playerTransform != nil && playerCollision.Grounded && playerCollision.GroundEntity == uint64(e)
		platform.DeltaX = 0
		platform.DeltaY = 0

		if platform.FramesUntilMove > 0 {
			platform.FramesUntilMove--
			if platform.FramesUntilMove == 0 && mode == component.MovingPlatformModeContinuous {
				platform.Moving = true
			}
		}

		if mode == component.MovingPlatformModeContinuous && platform.FramesUntilMove == 0 {
			platform.Moving = true
		}

		if mode == component.MovingPlatformModeTouch && !platform.Moving && platform.FramesUntilMove == 0 {
			if currentTouching {
				if !platform.TouchedWhileIdle {
					platform.Moving = true
					platform.TouchedWhileIdle = true
				}
			} else {
				platform.TouchedWhileIdle = false
			}
		}

		if !platform.Moving {
			applyMovingPlatformVelocity(w, e, 0, 0, false)
			return
		}

		nextX, nextY, deltaX, deltaY := stepMovingPlatform(platform, mode)
		platform.DeltaX = deltaX
		platform.DeltaY = deltaY

		if currentTouching && playerTransform != nil {
			if playerCollision != nil && playerPhysicsBody != nil && playerPhysicsBody.Body != nil {
				velocity := playerPhysicsBody.Body.Velocity()
				playerPhysicsBody.Body.SetVelocity(velocity.X+deltaX, velocity.Y+deltaY)
				playerCollision.GroundVelocityX = deltaX
				playerCollision.GroundVelocityY = deltaY
			} else {
				playerTransform.X += deltaX
				playerTransform.Y += deltaY
			}
		}

		applyMovingPlatformVelocity(w, e, deltaX, deltaY, true)
		if !movingPlatformUsesBody(w, e) {
			transform.X = nextX
			transform.Y = nextY
		}
	})
}

func normalizeMovingPlatformMode(mode component.MovingPlatformMode) component.MovingPlatformMode {
	// TODO - optimize by precomputing this when loading the level instead of every frame
	switch component.MovingPlatformMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case component.MovingPlatformModeTouch:
		return component.MovingPlatformModeTouch
	case component.MovingPlatformModeContinuous:
		return component.MovingPlatformModeContinuous
	default:
		return component.MovingPlatformModeContinuous
	}
}

func initializeMovingPlatform(platform *component.MovingPlatform, transform *component.Transform, mode component.MovingPlatformMode) {
	if platform == nil || transform == nil || platform.Initialized {
		return
	}
	platform.StartX = transform.X
	platform.StartY = transform.Y
	platform.Direction = 1
	platform.Progress = 0
	if platform.StartAtTarget {
		platform.Progress = 1
		platform.Direction = -1
		transform.X = platform.StartX + platform.DestX
		transform.Y = platform.StartY + platform.DestY
	}
	platform.Moving = mode == component.MovingPlatformModeContinuous
	platform.Initialized = true
}

func movingPlatformPosition(platform *component.MovingPlatform) (float64, float64) {
	if platform == nil {
		return 0, 0
	}
	return movingPlatformLerp(platform.StartX, platform.StartX+platform.DestX, platform.Progress), movingPlatformLerp(platform.StartY, platform.StartY+platform.DestY, platform.Progress)
}

func movingPlatformLerp(start, end, t float64) float64 {
	return start + (end-start)*t
}

func movingPlatformDirection(platform *component.MovingPlatform) float64 {
	if platform == nil {
		return 1
	}
	if platform.Direction < 0 {
		return -1
	}
	return 1
}

func stepMovingPlatform(platform *component.MovingPlatform, mode component.MovingPlatformMode) (float64, float64, float64, float64) {
	currentX, currentY := movingPlatformPosition(platform)
	distance := math.Hypot(platform.DestX, platform.DestY)
	if distance <= movingPlatformEpsilon || platform.Speed <= 0 {
		platform.Moving = false
		return currentX, currentY, 0, 0
	}

	step := platform.Speed / distance
	direction := movingPlatformDirection(platform)
	nextProgress := platform.Progress + direction*step
	reachedEnd := false
	if direction > 0 && nextProgress >= 1 {
		nextProgress = 1
		reachedEnd = true
	}
	if direction < 0 && nextProgress <= 0 {
		nextProgress = 0
		reachedEnd = true
	}
	platform.Progress = nextProgress
	nextX, nextY := movingPlatformPosition(platform)
	deltaX := nextX - currentX
	deltaY := nextY - currentY

	if reachedEnd {
		platform.Direction = -direction
		if mode == component.MovingPlatformModeTouch || platform.WaitFrames > 0 {
			platform.Moving = false
			platform.FramesUntilMove = platform.WaitFrames
		}
	}

	return nextX, nextY, deltaX, deltaY
}

func movingPlatformUsesBody(w *ecs.World, e ecs.Entity) bool {
	if w == nil {
		return false
	}
	body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind())
	return ok && body != nil && !body.Static && body.Body != nil
}

func applyMovingPlatformVelocity(w *ecs.World, e ecs.Entity, deltaX, deltaY float64, moving bool) {
	if w == nil {
		return
	}

	body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind())
	if !ok || body == nil || body.Static || body.Body == nil {
		return
	}

	if !moving || (math.Abs(deltaX) <= movingPlatformEpsilon && math.Abs(deltaY) <= movingPlatformEpsilon) {
		body.Body.SetVelocityVector(cp.Vector{})
		body.Body.SetAngularVelocity(0)
		return
	}

	body.Body.SetVelocityVector(cp.Vector{X: deltaX, Y: deltaY})
	body.Body.SetAngularVelocity(0)
}
