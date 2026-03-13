package component

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
)

type Sprite struct {
	Disabled   bool
	Image      *ebiten.Image
	Source     image.Rectangle
	UseSource  bool
	TileX      bool
	TileY      bool
	OriginX    float64
	OriginY    float64
	FacingLeft bool
}

var SpriteComponent = NewComponent[Sprite]()
