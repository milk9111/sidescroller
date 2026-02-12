package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type layerRenameDialog struct {
	Overlay *widget.Container
	Open    func(idx int, current string)
}

func newLayerRenameDialog(theme *widget.Theme, fontFace *text.Face, onLayerRenamed func(layerIndex int, newName string)) *layerRenameDialog {
	var renameIdx int = -1

	renameOverlay := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				StretchHorizontal:  true,
				StretchVertical:    true,
			}),
			widget.WidgetOpts.MinSize(1, 1),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{0, 0, 0, 160})),
	)
	renameOverlay.GetWidget().Visibility = widget.Visibility_Hide

	dialog := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(320, 140),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{220, 220, 220, 255})),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)
	dialog.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}

	nameLabel := widget.NewLabel(
		widget.LabelOpts.Text("Rename layer", fontFace, &widget.LabelColor{Idle: color.Black, Disabled: color.Gray{Y: 140}}),
	)
	nameInput := widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(260, 28),
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
		widget.TextInputOpts.SubmitOnEnter(true),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if renameIdx >= 0 && onLayerRenamed != nil && args.InputText != "" {
				onLayerRenamed(renameIdx, args.InputText)
			}
			renameOverlay.GetWidget().Visibility = widget.Visibility_Hide
			renameIdx = -1
		}),
	)

	buttonsRow := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)
	okBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("OK", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			text := nameInput.GetText()
			if renameIdx >= 0 && onLayerRenamed != nil && text != "" {
				onLayerRenamed(renameIdx, text)
			}
			renameOverlay.GetWidget().Visibility = widget.Visibility_Hide
			renameIdx = -1
		}),
	)
	cancelBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Cancel", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			renameOverlay.GetWidget().Visibility = widget.Visibility_Hide
			renameIdx = -1
		}),
	)
	buttonsRow.AddChild(okBtn)
	buttonsRow.AddChild(cancelBtn)

	dialog.AddChild(nameLabel)
	dialog.AddChild(nameInput)
	dialog.AddChild(buttonsRow)
	renameOverlay.AddChild(dialog)

	open := func(idx int, current string) {
		renameIdx = idx
		nameInput.SetText(current)
		nameInput.Focus(true)
		renameOverlay.GetWidget().Visibility = widget.Visibility_Show
	}

	return &layerRenameDialog{Overlay: renameOverlay, Open: open}
}
