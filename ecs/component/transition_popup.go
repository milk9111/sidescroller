package component

import "github.com/hajimehoshi/ebiten/v2"

type TransitionPopup struct {
	KeyboardCue            *ebiten.Image
	GamepadCue             *ebiten.Image
	Base                   *ebiten.Image
	TargetTransitionEntity uint64
	HasRenderedImage       bool
	RenderedGamepad        bool
}

var TransitionPopupComponent = NewComponent[TransitionPopup]()
