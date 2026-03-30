package component

import "github.com/hajimehoshi/ebiten/v2"

type ItemPopup struct {
	KeyboardCue      *ebiten.Image
	GamepadCue       *ebiten.Image
	Base             *ebiten.Image
	TargetItemEntity uint64
	HasRenderedImage bool
	RenderedGamepad  bool
}

var ItemPopupComponent = NewComponent[ItemPopup]()
