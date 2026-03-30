package component

import "github.com/hajimehoshi/ebiten/v2"

type Item struct {
	Description string
	Range       float64
	Image       *ebiten.Image
}

var ItemComponent = NewComponent[Item]()
