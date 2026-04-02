package scenes

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestConsumeLevelLoadWarmupRunsOncePerSequence(t *testing.T) {
	w := ecs.NewWorld()
	loadedEnt := ecs.CreateEntity(w)
	if err := ecs.Add(w, loadedEnt, component.LevelLoadedComponent.Kind(), &component.LevelLoaded{Sequence: 1}); err != nil {
		t.Fatalf("add level loaded marker: %v", err)
	}

	var (
		lastSequence uint64
		calls        int
	)
	warmup := func(*ecs.World) {
		calls++
	}

	if !consumeLevelLoadWarmup(w, &lastSequence, warmup) {
		t.Fatal("expected warmup to run for the first load sequence")
	}
	if calls != 1 {
		t.Fatalf("expected one warmup call, got %d", calls)
	}
	if lastSequence != 1 {
		t.Fatalf("expected last sequence 1, got %d", lastSequence)
	}

	if consumeLevelLoadWarmup(w, &lastSequence, warmup) {
		t.Fatal("expected duplicate warmup to be skipped for the same load sequence")
	}
	if calls != 1 {
		t.Fatalf("expected warmup call count to stay at 1, got %d", calls)
	}

	loaded, ok := ecs.Get(w, loadedEnt, component.LevelLoadedComponent.Kind())
	if !ok || loaded == nil {
		t.Fatal("expected level loaded marker to remain available")
	}
	loaded.Sequence = 2

	if !consumeLevelLoadWarmup(w, &lastSequence, warmup) {
		t.Fatal("expected warmup to run for the next load sequence")
	}
	if calls != 2 {
		t.Fatalf("expected two warmup calls after a new sequence, got %d", calls)
	}
	if lastSequence != 2 {
		t.Fatalf("expected last sequence 2, got %d", lastSequence)
	}
}

func TestClearQueuedPlayerAudioResetsPlayerFlags(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.AudioComponent.Kind(), &component.Audio{
		Names: []string{"land", "run"},
		Play:  []bool{true, false},
		Stop:  []bool{false, true},
	}); err != nil {
		t.Fatalf("add player audio: %v", err)
	}

	clearQueuedPlayerAudio(w)

	audioComp, ok := ecs.Get(w, player, component.AudioComponent.Kind())
	if !ok || audioComp == nil {
		t.Fatal("expected player audio component")
	}
	for i, queued := range audioComp.Play {
		if queued {
			t.Fatalf("expected play flag %d to be cleared", i)
		}
	}
	for i, queued := range audioComp.Stop {
		if queued {
			t.Fatalf("expected stop flag %d to be cleared", i)
		}
	}
}