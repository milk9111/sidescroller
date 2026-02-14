package component

import "github.com/hajimehoshi/ebiten/v2/audio"

type Audio struct {
	Names   []string
	Players []*audio.Player
	Volume  []float64
	Play    []bool
	Stop    []bool
}

var AudioComponent = NewComponent[Audio]()
