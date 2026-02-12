package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func buildToolBar(theme *widget.Theme, fontFace *text.Face, onToolSelected func(tool Tool), initialTool Tool) (*widget.Container, *ToolBar) {
	toolNames := []string{"Brush", "Erase", "Fill", "Line"}
	buttonTextColor := &widget.ButtonTextColor{
		Idle:     color.Black,
		Hover:    color.Black,
		Pressed:  color.RGBA{0, 0, 200, 255},
		Disabled: color.Gray{Y: 128},
	}

	toolbar := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(220, 48),
		),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{220, 220, 240, 255})),
	)

	var toolButtons []*widget.Button
	for _, name := range toolNames {
		btn := widget.NewButton(
			widget.ButtonOpts.Image(theme.ButtonTheme.Image),
			widget.ButtonOpts.Text(name, fontFace, buttonTextColor),
			widget.ButtonOpts.ToggleMode(),
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(48, 40),
			),
		)
		toolButtons = append(toolButtons, btn)
		toolbar.AddChild(btn)
	}

	elements := make([]widget.RadioGroupElement, 0, len(toolButtons))
	for _, b := range toolButtons {
		elements = append(elements, b)
	}

	group := widget.NewRadioGroup(
		widget.RadioGroupOpts.Elements(elements...),
		widget.RadioGroupOpts.ChangedHandler(func(args *widget.RadioGroupChangedEventArgs) {
			if onToolSelected == nil {
				return
			}
			for idx, b := range toolButtons {
				if args.Active == b {
					onToolSelected(Tool(idx))
					return
				}
			}
		}),
	)

	if idx := int(initialTool); idx >= 0 && idx < len(toolButtons) {
		group.SetActive(toolButtons[idx])
	}

	return toolbar, &ToolBar{group: group, buttons: toolButtons}
}
