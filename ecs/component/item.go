package component

import "github.com/hajimehoshi/ebiten/v2"

type Item struct {
	Prefab      string
	Description string
	Range       float64
	Image       *ebiten.Image
}

var ItemComponent = NewComponent[Item]()
