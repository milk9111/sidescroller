package entity

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func NewMusicPlayer(w *ecs.World) (ecs.Entity, error) {
	ent, err := BuildEntity(w, "music_player.yaml")
	if err != nil {
		return 0, fmt.Errorf("music player: %w", err)
	}
	return ent, nil
}

func CloneMusicPlayerState(src *component.MusicPlayer) *component.MusicPlayer {
	if src == nil {
		return nil
	}

	players := make(map[string]*audio.Player, len(src.Players))
	for track, player := range src.Players {
		players[track] = player
	}

	trackVolumes := make(map[string]float64, len(src.TrackVolumes))
	for track, volume := range src.TrackVolumes {
		trackVolumes[track] = volume
	}

	return &component.MusicPlayer{
		Players:       players,
		TrackVolumes:  trackVolumes,
		CurrentTrack:  src.CurrentTrack,
		CurrentVolume: src.CurrentVolume,
		CurrentLoop:   src.CurrentLoop,
		PendingTrack:  src.PendingTrack,
		PendingVolume: src.PendingVolume,
		PendingLoop:   src.PendingLoop,
		PendingActive: src.PendingActive,
		FadeStep:      src.FadeStep,
	}
}

func NewMusicPlayerFromState(w *ecs.World, state *component.MusicPlayer) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("music player: world is nil")
	}
	if state == nil {
		return NewMusicPlayer(w)
	}

	ent := ecs.CreateEntity(w)
	if err := ecs.Add(w, ent, component.MusicPlayerComponent.Kind(), CloneMusicPlayerState(state)); err != nil {
		return 0, fmt.Errorf("music player: add component: %w", err)
	}
	return ent, nil
}
