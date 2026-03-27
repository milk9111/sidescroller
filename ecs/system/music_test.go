package system

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestMusicSystemMutedConsumesRequestWithoutPlaying(t *testing.T) {
	trackPlayer, err := assets.LoadAudioPlayer("boss_music.wav")
	if err != nil {
		t.Fatalf("load music player: %v", err)
	}

	w := ecs.NewWorld()
	musicEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, musicEntity, component.MusicPlayerComponent.Kind(), &component.MusicPlayer{
		Players: map[string]*audio.Player{
			"boss_music.wav": trackPlayer,
		},
	}); err != nil {
		t.Fatalf("add music player: %v", err)
	}

	requestEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, requestEntity, component.MusicRequestComponent.Kind(), &component.MusicRequest{
		Track:         "boss_music.wav",
		Volume:        0.6,
		Loop:          true,
		FadeOutFrames: 15,
	}); err != nil {
		t.Fatalf("add music request: %v", err)
	}

	NewMusicSystem(true).Update(w)

	player, ok := ecs.Get(w, musicEntity, component.MusicPlayerComponent.Kind())
	if !ok || player == nil {
		t.Fatal("expected music player component")
	}
	if player.CurrentTrack != "boss_music.wav" {
		t.Fatalf("expected current track to be updated, got %q", player.CurrentTrack)
	}
	if trackPlayer.IsPlaying() {
		t.Fatal("expected muted music system to avoid playback")
	}
	if _, ok := ecs.Get(w, requestEntity, component.MusicRequestComponent.Kind()); ok {
		t.Fatal("expected request entity to be consumed")
	}
}
