package system

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func addGameplayTimeScale(t *testing.T, w *ecs.World, scale float64) {
	t.Helper()

	e := ecs.CreateEntity(w)
	if err := ecs.Add(w, e, component.GameplayTimeComponent.Kind(), &component.GameplayTime{Scale: scale}); err != nil {
		t.Fatalf("add gameplay time: %v", err)
	}
}

func simulatePhysicsDeltaX(t *testing.T, scale float64) float64 {
	t.Helper()

	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	if err := ecs.Add(w, e, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transform: %v", err)
	}
	if err := ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 20, Height: 20, Mass: 1}); err != nil {
		t.Fatalf("add physics body: %v", err)
	}
	addGameplayTimeScale(t, w, scale)

	ps := NewPhysicsSystem()
	ps.Update(w)

	body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind())
	if !ok || body == nil || body.Body == nil {
		t.Fatal("expected physics body to be created")
	}
	body.Body.SetVelocityVector(cp.Vector{X: 8, Y: 0})

	transform, ok := ecs.Get(w, e, component.TransformComponent.Kind())
	if !ok || transform == nil {
		t.Fatal("expected transform")
	}
	startX := transform.X

	ps.Update(w)

	return transform.X - startX
}

func TestPhysicsSystemScalesGameplayStep(t *testing.T) {
	normalDelta := simulatePhysicsDeltaX(t, 1)
	slowDelta := simulatePhysicsDeltaX(t, 0.25)

	if slowDelta >= normalDelta {
		t.Fatalf("expected slow gameplay delta %.3f to be smaller than normal delta %.3f", slowDelta, normalDelta)
	}

	want := normalDelta * 0.25
	if math.Abs(slowDelta-want) > 0.15 {
		t.Fatalf("expected slow gameplay delta close to %.3f, got %.3f", want, slowDelta)
	}
}

func TestAnimationSystemScalesFrameAdvance(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	addGameplayTimeScale(t, w, 0.25)

	anim := &component.Animation{
		Sheet:   ebiten.NewImage(16, 4),
		Current: "run",
		Defs: map[string]component.AnimationDef{
			"run": {
				Name:       "run",
				FrameCount: 2,
				FrameW:     8,
				FrameH:     4,
				FPS:        30,
				Loop:       true,
			},
		},
		Playing: true,
	}
	if err := ecs.Add(w, e, component.AnimationComponent.Kind(), anim); err != nil {
		t.Fatalf("add animation: %v", err)
	}
	if err := ecs.Add(w, e, component.SpriteComponent.Kind(), &component.Sprite{}); err != nil {
		t.Fatalf("add sprite: %v", err)
	}

	system := NewAnimationSystem()
	for range 7 {
		system.Update(w)
	}

	if anim.Frame != 0 {
		t.Fatalf("expected animation frame to remain at 0 before the slowed threshold, got %d", anim.Frame)
	}

	system.Update(w)
	if anim.Frame != 1 {
		t.Fatalf("expected animation to advance on the eighth slowed update, got frame %d", anim.Frame)
	}
	if anim.FrameTimer != 0 {
		t.Fatalf("expected animation timer to wrap to 0 after advancing, got %d", anim.FrameTimer)
	}
}
