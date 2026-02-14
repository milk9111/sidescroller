package component

import "github.com/hajimehoshi/ebiten/v2/audio"

type Audio struct {
	Names   []string
	Players []*audio.Player
	Play    []bool
}

var AudioComponent = NewComponent[Audio]()
