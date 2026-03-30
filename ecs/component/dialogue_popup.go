package component

import "github.com/hajimehoshi/ebiten/v2"

type DialoguePopup struct {
	KeyboardCue          *ebiten.Image
	GamepadCue           *ebiten.Image
	Base                 *ebiten.Image
	TargetDialogueEntity uint64
	HasRenderedImage     bool
	RenderedGamepad      bool
}

var DialoguePopupComponent = NewComponent[DialoguePopup]()
