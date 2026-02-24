package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func newGateSection(
	parent *widget.Container,
	theme *widget.Theme,
	fontFace *text.Face,
	onToggleGateMode func(enabled bool),
) *GateUI {
	gateUI := &GateUI{}

	gateLabel := widget.NewLabel(
		widget.LabelOpts.Text("Gates", fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	parent.AddChild(gateLabel)

	gateModeBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Gates: Off", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			gateUI.modeOn = !gateUI.modeOn
			gateUI.SetMode(gateUI.modeOn)
			if onToggleGateMode != nil {
				onToggleGateMode(gateUI.modeOn)
			}
		}),
	)
	gateUI.modeBtn = gateModeBtn
	parent.AddChild(gateModeBtn)

	return gateUI
}
