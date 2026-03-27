package system

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func testAudioPlayer(t *testing.T) *audio.Player {
	t.Helper()

	player, err := assets.LoadAudioPlayer("player_jump.wav")
	if err != nil {
		t.Fatalf("load audio player: %v", err)
	}
	return player
}

func TestAudioSystemMutedSuppressesPlayback(t *testing.T) {
	w := ecs.NewWorld()
	ent := ecs.CreateEntity(w)
	player := testAudioPlayer(t)

	if err := ecs.Add(w, ent, component.AudioComponent.Kind(), &component.Audio{
		Players: []*audio.Player{player},
		Volume:  []float64{1},
		Play:    []bool{true},
		Stop:    []bool{false},
	}); err != nil {
		t.Fatalf("add audio component: %v", err)
	}

	NewAudioSystem(true).Update(w)

	audioComp, ok := ecs.Get(w, ent, component.AudioComponent.Kind())
	if !ok || audioComp == nil {
		t.Fatal("expected audio component")
	}
	if audioComp.Play[0] {
		t.Fatal("expected muted audio system to clear play request")
	}
	if player.IsPlaying() {
		t.Fatal("expected muted audio system to suppress playback")
	}
}

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
