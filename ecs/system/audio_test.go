package system

import (
	"math"
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestAudioVolumeForEntityUsesFullVolumeNearPlayer(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 100, Y: 100}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}

	emitter := ecs.CreateEntity(w)
	if err := ecs.Add(w, emitter, component.TransformComponent.Kind(), &component.Transform{X: 140, Y: 120}); err != nil {
		t.Fatalf("add emitter transform: %v", err)
	}

	got := audioVolumeForEntity(w, emitter, 0.75)
	if math.Abs(got-0.75) > 1e-9 {
		t.Fatalf("expected full base volume near player, got %f", got)
	}
}

func TestAudioVolumeForEntityAttenuatesFarEmitter(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}

	emitter := ecs.CreateEntity(w)
	if err := ecs.Add(w, emitter, component.TransformComponent.Kind(), &component.Transform{X: audioFalloffMaxDistance + 250, Y: 0}); err != nil {
		t.Fatalf("add emitter transform: %v", err)
	}

	got := audioVolumeForEntity(w, emitter, 1)
	if math.Abs(got-audioMinDistanceVolume) > 1e-9 {
		t.Fatalf("expected min attenuated volume %f, got %f", audioMinDistanceVolume, got)
	}
}

func TestAudioVolumeForEntityWithoutPlayerKeepsBaseVolume(t *testing.T) {
	w := ecs.NewWorld()

	emitter := ecs.CreateEntity(w)
	if err := ecs.Add(w, emitter, component.TransformComponent.Kind(), &component.Transform{X: 400, Y: 200}); err != nil {
		t.Fatalf("add emitter transform: %v", err)
	}

	got := audioVolumeForEntity(w, emitter, 0.4)
	if math.Abs(got-0.4) > 1e-9 {
		t.Fatalf("expected unchanged base volume without player, got %f", got)
	}
}

func TestEntityWorldPositionPrefersPhysicsBody(t *testing.T) {
	w := ecs.NewWorld()

	ent := ecs.CreateEntity(w)
	if err := ecs.Add(w, ent, component.TransformComponent.Kind(), &component.Transform{X: 10, Y: 20, Parent: 99, WorldX: 30, WorldY: 40}); err != nil {
		t.Fatalf("add transform: %v", err)
	}
	if err := ecs.Add(w, ent, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{}); err != nil {
		t.Fatalf("add physics body: %v", err)
	}

	x, y, ok := entityWorldPosition(w, ent)
	if !ok {
		t.Fatal("expected transform fallback position")
	}
	if x != 30 || y != 40 {
		t.Fatalf("expected world transform fallback (30, 40), got (%f, %f)", x, y)
	}
}
