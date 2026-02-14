package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AudioSystem struct{}

func NewAudioSystem() *AudioSystem {
	return &AudioSystem{}
}

func (a *AudioSystem) Update(w *ecs.World) {
	ecs.ForEach(w, component.AudioComponent.Kind(), func(_ ecs.Entity, audioComp *component.Audio) {
		count := len(audioComp.Play)
		if len(audioComp.Players) < count {
			count = len(audioComp.Players)
		}

		for i := 0; i < count; i++ {
			if !audioComp.Play[i] {
				continue
			}

			player := audioComp.Players[i]
			if player != nil && !player.IsPlaying() {
				player.SetVolume(audioComp.Volume[i])
				player.Rewind()
				player.Play()
			}

			audioComp.Play[i] = false
		}

		for i := 0; i < count; i++ {
			if !audioComp.Stop[i] {
				continue
			}

			player := audioComp.Players[i]
			if player != nil && player.IsPlaying() {
				player.Pause()
			}

			audioComp.Stop[i] = false
		}
	})
}
