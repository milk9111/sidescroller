package component

import "github.com/hajimehoshi/ebiten/v2"

type Dialogue struct {
	Lines    []string
	Range    float64
	Portrait *ebiten.Image
}

var DialogueComponent = NewComponent[Dialogue]()
