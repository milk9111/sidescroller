package entity

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func buildAudioComponent(audioSpecs []prefabs.AudioSpec) (*component.Audio, error) {
	n := len(audioSpecs)
	if n == 0 {
		return nil, nil
	}

	names := make([]string, 0, n)
	players := make([]*audio.Player, 0, n)
	play := make([]bool, 0, n)

	for i, clip := range audioSpecs {
		player, err := assets.LoadAudioPlayer(clip.File)
		if err != nil {
			return nil, fmt.Errorf("audio clip %d (%q): %w", i, clip.Name, err)
		}
		names = append(names, clip.Name)
		players = append(players, player)
		play = append(play, false)
	}

	return &component.Audio{
		Names:   names,
		Players: players,
		Play:    play,
	}, nil
}
