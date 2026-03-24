package component

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
)

type UIRoot struct {
	UI *ebitenui.UI
}

var UIRootComponent = NewComponent[UIRoot]()

type DialogueUI struct {
	Root        *widget.Container
	Overlay     *widget.Container
	Panel       *widget.Container
	PortraitBox *widget.Container
	Portrait    *widget.Graphic
	Text        *widget.Text
}

var DialogueUIComponent = NewComponent[DialogueUI]()

type DialogueState struct {
	Active         bool
	DialogueEntity uint64
	LineIndex      int
}

var DialogueStateComponent = NewComponent[DialogueState]()
