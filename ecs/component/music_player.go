package component

import "github.com/hajimehoshi/ebiten/v2/audio"

// MusicPlayer stores global music playback state on a dedicated ECS entity.
// The music system mutates this component; no playback state is kept on the system.
type MusicPlayer struct {
	Players      map[string]*audio.Player
	TrackVolumes map[string]float64

	CurrentTrack  string
	CurrentVolume float64
	CurrentLoop   bool

	PendingTrack  string
	PendingVolume float64
	PendingLoop   bool
	PendingActive bool

	FadeStep float64
}

var MusicPlayerComponent = NewComponent[MusicPlayer]()
