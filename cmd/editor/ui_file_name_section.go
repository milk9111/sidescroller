package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func addFileNameSection(parent *widget.Container, fontFace *text.Face) *widget.TextInput {
	fileLabel := widget.NewLabel(
		widget.LabelOpts.Text("File", fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	fileNameInput := widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(180, 28),
		),
		widget.TextInputOpts.Image(&widget.TextInputImage{
			Idle:     solidNineSlice(color.RGBA{245, 245, 245, 255}),
			Disabled: solidNineSlice(color.RGBA{200, 200, 200, 255}),
		}),
		widget.TextInputOpts.Color(&widget.TextInputColor{
			Idle:     color.Black,
			Disabled: color.Gray{Y: 120},
			Caret:    color.Black,
		}),
		widget.TextInputOpts.Face(fontFace),
	)
	parent.AddChild(fileLabel)
	parent.AddChild(fileNameInput)
	return fileNameInput
}
