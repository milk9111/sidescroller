package component

import "github.com/hajimehoshi/ebiten/v2"

type ShrinePopup struct {
	KeyboardCue      *ebiten.Image
	GamepadCue       *ebiten.Image
	Base             *ebiten.Image
	TargetShrineEntity uint64
	HasRenderedImage bool
	RenderedGamepad  bool
}

var ShrinePopupComponent = NewComponent[ShrinePopup]()