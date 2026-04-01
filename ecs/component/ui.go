package component

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type UIRoot struct {
	UI *ebitenui.UI
}

var UIRootComponent = NewComponent[UIRoot]()

type DialogueUI struct {
	Root         *widget.Container
	HUDLayer     *widget.Container
	OverlayLayer *widget.Container
	Overlay      *widget.Container
	Panel        *widget.Container
	PortraitBox  *widget.Container
	Portrait     *widget.Graphic
	Text         *widget.Text
}

var DialogueUIComponent = NewComponent[DialogueUI]()

type DialogueState struct {
	Active         bool
	DialogueEntity uint64
	LineIndex      int
}

var DialogueStateComponent = NewComponent[DialogueState]()

type ItemUI struct {
	Root    *widget.Container
	Overlay *widget.Container
	Panel   *widget.Container
	Image   *widget.Graphic
	Text    *widget.Text
}

var ItemUIComponent = NewComponent[ItemUI]()

type ItemState struct {
	Active     bool
	ItemEntity uint64
}

var ItemStateComponent = NewComponent[ItemState]()

type InventoryUI struct {
	Root        *widget.Container
	Overlay     *widget.Container
	Panel       *widget.Container
	Title       *widget.Text
	GridHost    *widget.Container
	DetailPanel *widget.Container
	DetailImage *widget.Graphic
	DetailText  *widget.Text
}

var InventoryUIComponent = NewComponent[InventoryUI]()

type InventoryState struct {
	Active        bool
	SelectedIndex int
	LastMoveX     int
	LastMoveY     int
}

var InventoryStateComponent = NewComponent[InventoryState]()

type PlayerHUDUI struct {
	Root            *widget.Container
	Hearts          []*widget.Graphic
	HeartFullImage  *ebiten.Image
	HeartEmptyImage *ebiten.Image
	GearText        *widget.Text
}

var PlayerHUDUIComponent = NewComponent[PlayerHUDUI]()
