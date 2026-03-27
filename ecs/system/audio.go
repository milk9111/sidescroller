package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AudioSystem struct {
	muted bool
}

const (
	audioFullVolumeDistance = 96.0
	audioFalloffMaxDistance = 960.0
	audioMinDistanceVolume  = 0.08
)

func NewAudioSystem(muted bool) *AudioSystem {
	return &AudioSystem{muted: muted}
}

func (a *AudioSystem) Update(w *ecs.World) {
	ecs.ForEach(w, component.AudioComponent.Kind(), func(e ecs.Entity, audioComp *component.Audio) {
		count := len(audioComp.Play)
		if len(audioComp.Players) < count {
			count = len(audioComp.Players)
		}

		if a.muted {
			for i := 0; i < len(audioComp.Players); i++ {
				player := audioComp.Players[i]
				if player != nil && player.IsPlaying() {
					player.Pause()
				}
			}
			for i := 0; i < len(audioComp.Play); i++ {
				audioComp.Play[i] = false
			}
			for i := 0; i < len(audioComp.Stop); i++ {
				audioComp.Stop[i] = false
			}
			return
		}

		for i := 0; i < count; i++ {
			if !audioComp.Play[i] {
				continue
			}

			player := audioComp.Players[i]
			if player != nil && !player.IsPlaying() {
				player.SetVolume(audioVolumeForEntity(w, e, audioComp.Volume[i]))
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

func audioVolumeForEntity(w *ecs.World, ent ecs.Entity, baseVolume float64) float64 {
	if baseVolume <= 0 {
		return 0
	}

	listenerX, listenerY, ok := playerWorldPosition(w)
	if !ok {
		return baseVolume
	}

	emitterX, emitterY, ok := entityWorldPosition(w, ent)
	if !ok {
		return baseVolume
	}

	dx := emitterX - listenerX
	dy := emitterY - listenerY
	distance := math.Hypot(dx, dy)

	mult := audioDistanceMultiplier(distance)

	return baseVolume * mult
}

func audioDistanceMultiplier(distance float64) float64 {
	if distance <= audioFullVolumeDistance {
		return 1
	}
	if distance >= audioFalloffMaxDistance {
		return audioMinDistanceVolume
	}

	rangeSpan := audioFalloffMaxDistance - audioFullVolumeDistance
	if rangeSpan <= 0 {
		return 1
	}

	t := (distance - audioFullVolumeDistance) / rangeSpan
	return 1 - t*(1-audioMinDistanceVolume)
}
