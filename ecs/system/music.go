package system

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	defaultMusicVolume     = 1.0
	defaultMusicFadeFrames = 30
)

type MusicSystem struct{}

func NewMusicSystem() *MusicSystem {
	return &MusicSystem{}
}

func RequestMusic(w *ecs.World, track string) {
	RequestMusicWithOptions(w, &component.MusicRequest{Track: track, Volume: 0, Loop: true, FadeOutFrames: defaultMusicFadeFrames})
}

func RequestMusicWithOptions(w *ecs.World, req *component.MusicRequest) {
	if w == nil || req == nil {
		return
	}
	ent := ecs.CreateEntity(w)
	_ = ecs.Add(w, ent, component.MusicRequestComponent.Kind(), req)
}

func StopMusic(w *ecs.World) {
	RequestMusicWithOptions(w, &component.MusicRequest{FadeOutFrames: defaultMusicFadeFrames})
}

func (m *MusicSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	latest, requestEntities := m.consumeLatestRequest(w)
	for _, ent := range requestEntities {
		ecs.DestroyEntity(w, ent)
	}

	ent, ok := ecs.First(w, component.MusicPlayerComponent.Kind())
	if !ok {
		return
	}
	player, ok := ecs.Get(w, ent, component.MusicPlayerComponent.Kind())
	if !ok || player == nil {
		return
	}
	if player.Players == nil {
		player.Players = make(map[string]*audio.Player)
	}
	if player.TrackVolumes == nil {
		player.TrackVolumes = make(map[string]float64)
	}

	if latest != nil {
		m.applyRequest(player, *latest)
	}

	if player.PendingActive {
		m.updateTransition(player)
		return
	}

	currentPlayer := m.currentPlayer(player)
	if currentPlayer != nil && !currentPlayer.IsPlaying() && player.CurrentTrack != "" && player.CurrentLoop {
		currentPlayer.Rewind()
		currentPlayer.SetVolume(player.CurrentVolume)
		currentPlayer.Play()
	}
}
func (m *MusicSystem) consumeLatestRequest(w *ecs.World) (*component.MusicRequest, []ecs.Entity) {
	var latest *component.MusicRequest
	requestEntities := make([]ecs.Entity, 0)

	ecs.ForEach(w, component.MusicRequestComponent.Kind(), func(ent ecs.Entity, req *component.MusicRequest) {
		requestEntities = append(requestEntities, ent)
		if req == nil {
			return
		}
		copy := *req
		latest = &copy
	})

	return latest, requestEntities
}

func (m *MusicSystem) applyRequest(player *component.MusicPlayer, req component.MusicRequest) {
	if player == nil {
		return
	}

	track := strings.TrimSpace(req.Track)
	volume := req.Volume
	if volume <= 0 {
		if v, ok := player.TrackVolumes[track]; ok && v > 0 {
			volume = v
		} else {
			volume = defaultMusicVolume
		}
	}
	if volume > 1 {
		volume = 1
	}
	loop := req.Loop
	fadeFrames := req.FadeOutFrames
	if fadeFrames <= 0 {
		fadeFrames = defaultMusicFadeFrames
	}

	if track == "" {
		player.PendingActive = false
		if m.currentPlayer(player) == nil {
			player.CurrentTrack = ""
			player.CurrentVolume = 0
			player.CurrentLoop = false
			return
		}
		player.PendingTrack = ""
		player.PendingVolume = 0
		player.PendingLoop = false
		player.PendingActive = true
		player.FadeStep = player.CurrentVolume / float64(fadeFrames)
		if player.FadeStep <= 0 {
			player.FadeStep = 1
		}
		return
	}

	currentPlayer := m.currentPlayer(player)
	if !player.PendingActive && player.CurrentTrack == track && currentPlayer != nil {
		player.CurrentVolume = volume
		currentPlayer.SetVolume(player.CurrentVolume)
		if !currentPlayer.IsPlaying() {
			currentPlayer.Rewind()
			currentPlayer.Play()
		}
		return
	}

	player.PendingTrack = track
	player.PendingVolume = volume
	player.PendingLoop = loop
	player.PendingActive = true
	if currentPlayer == nil {
		m.switchToPending(player)
		return
	}

	player.FadeStep = player.CurrentVolume / float64(fadeFrames)
	if player.FadeStep <= 0 {
		player.FadeStep = 1
	}
}

func (m *MusicSystem) updateTransition(player *component.MusicPlayer) {
	if player == nil {
		return
	}

	currentPlayer := m.currentPlayer(player)
	if currentPlayer == nil {
		m.switchToPending(player)
		return
	}

	player.CurrentVolume -= player.FadeStep
	if player.CurrentVolume > 0 {
		currentPlayer.SetVolume(player.CurrentVolume)
		return
	}

	player.CurrentVolume = 0
	currentPlayer.SetVolume(0)
	currentPlayer.Pause()
	currentPlayer.Rewind()
	player.CurrentTrack = ""
	player.CurrentLoop = false
	m.switchToPending(player)
}

func (m *MusicSystem) switchToPending(player *component.MusicPlayer) {
	if player == nil || !player.PendingActive {
		return
	}

	reqTrack := strings.TrimSpace(player.PendingTrack)
	reqVolume := player.PendingVolume
	reqLoop := player.PendingLoop

	player.PendingTrack = ""
	player.PendingVolume = 0
	player.PendingLoop = false
	player.PendingActive = false
	player.FadeStep = 0

	if reqTrack == "" {
		player.CurrentTrack = ""
		player.CurrentVolume = 0
		player.CurrentLoop = false
		return
	}

	audioPlayer, err := m.playerForTrack(player, reqTrack)
	if err != nil {
		fmt.Printf("music: load %q: %v\n", reqTrack, err)
		player.CurrentTrack = ""
		player.CurrentVolume = 0
		player.CurrentLoop = false
		return
	}

	player.CurrentTrack = reqTrack
	player.CurrentVolume = reqVolume
	player.CurrentLoop = reqLoop
	audioPlayer.Rewind()
	audioPlayer.SetVolume(player.CurrentVolume)
	audioPlayer.Play()
}

func (m *MusicSystem) currentPlayer(player *component.MusicPlayer) *audio.Player {
	if player == nil || strings.TrimSpace(player.CurrentTrack) == "" || player.Players == nil {
		return nil
	}
	audioPlayer, ok := player.Players[player.CurrentTrack]
	if !ok {
		return nil
	}
	return audioPlayer
}

func (m *MusicSystem) playerForTrack(player *component.MusicPlayer, track string) (*audio.Player, error) {
	if player == nil {
		return nil, fmt.Errorf("music player component is nil")
	}
	if player.Players == nil {
		player.Players = make(map[string]*audio.Player)
	}

	if existing, ok := player.Players[track]; ok && existing != nil {
		return existing, nil
	}

	audioPlayer, err := assets.LoadAudioPlayer(track)
	if err != nil {
		return nil, err
	}
	player.Players[track] = audioPlayer
	return audioPlayer, nil
}
