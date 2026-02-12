package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func newTransitionSection(
	parent *widget.Container,
	theme *widget.Theme,
	fontFace *text.Face,
	onToggleTransitionMode func(enabled bool),
	onTransitionFieldChanged func(field, value string),
) *TransitionUI {
	transitionUI := &TransitionUI{}
	transitionUI.relayoutTarget = parent

	transitionLabel := widget.NewLabel(
		widget.LabelOpts.Text("Transitions", fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	parent.AddChild(transitionLabel)

	transitionModeBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Transitions: Off", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			transitionUI.modeOn = !transitionUI.modeOn
			transitionUI.SetMode(transitionUI.modeOn)
			if onToggleTransitionMode != nil {
				onToggleTransitionMode(transitionUI.modeOn)
			}
		}),
	)
	transitionUI.modeBtn = transitionModeBtn

	transitionForm := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(6),
			),
		),
	)
	transitionForm.GetWidget().Visibility = widget.Visibility_Hide
	transitionUI.form = transitionForm

	transitionContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(4),
			),
		),
	)
	transitionContainer.AddChild(transitionModeBtn)
	transitionContainer.AddChild(transitionForm)
	parent.AddChild(transitionContainer)

	makeField := func(label string, onChange func(text string)) *widget.TextInput {
		transitionForm.AddChild(widget.NewLabel(
			widget.LabelOpts.Text(label, fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
		))
		input := widget.NewTextInput(
			widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.MinSize(180, 28)),
			widget.TextInputOpts.Image(&widget.TextInputImage{
				Idle:     solidNineSlice(color.RGBA{245, 245, 245, 255}),
				Disabled: solidNineSlice(color.RGBA{200, 200, 200, 255}),
			}),
			widget.TextInputOpts.Color(&widget.TextInputColor{Idle: color.Black, Disabled: color.Gray{Y: 120}, Caret: color.Black}),
			widget.TextInputOpts.Face(fontFace),
			widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
				if transitionUI.suppress {
					return
				}
				if onChange != nil {
					onChange(args.InputText)
				}
			}),
		)
		transitionForm.AddChild(input)
		return input
	}

	transitionUI.idInput = makeField("ID", func(text string) {
		if onTransitionFieldChanged != nil {
			onTransitionFieldChanged("id", text)
		}
	})
	transitionUI.levelInput = makeField("To level", func(text string) {
		if onTransitionFieldChanged != nil {
			onTransitionFieldChanged("to_level", text)
		}
	})
	transitionUI.linkedInput = makeField("Linked transition ID", func(text string) {
		if onTransitionFieldChanged != nil {
			onTransitionFieldChanged("linked_id", text)
		}
	})

	transitionForm.AddChild(widget.NewLabel(
		widget.LabelOpts.Text("Enter direction", fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	))

	// Radio buttons for enter direction
	dirLabels := []string{"up", "down", "left", "right"}
	var dirButtons []*widget.Button
	dirRow := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(6),
			),
		),
	)
	for _, lbl := range dirLabels {
		lblCopy := lbl
		btn := widget.NewButton(
			widget.ButtonOpts.Image(theme.ButtonTheme.Image),
			widget.ButtonOpts.Text(lblCopy, fontFace, theme.ButtonTheme.TextColor),
			widget.ButtonOpts.ToggleMode(),
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(80, 28)),
			widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
				if transitionUI.suppress {
					return
				}
				if onTransitionFieldChanged != nil {
					val := lblCopy
					onTransitionFieldChanged("enter_dir", val)
				}
			}),
		)
		dirButtons = append(dirButtons, btn)
		dirRow.AddChild(btn)
	}
	transitionForm.AddChild(dirRow)

	elements := make([]widget.RadioGroupElement, 0, len(dirButtons))
	for _, b := range dirButtons {
		elements = append(elements, b)
	}
	dirGroup := widget.NewRadioGroup(
		widget.RadioGroupOpts.Elements(elements...),
		widget.RadioGroupOpts.ChangedHandler(func(args *widget.RadioGroupChangedEventArgs) {
			if transitionUI.suppress {
				return
			}
			if onTransitionFieldChanged != nil {
				for i, b := range dirButtons {
					if args.Active == b {
						val := dirLabels[i]
						if val == "(none)" {
							val = ""
						}
						onTransitionFieldChanged("enter_dir", val)
						return
					}
				}
			}
		}),
	)
	transitionUI.dirGroup = dirGroup
	transitionUI.dirButtons = dirButtons

	return transitionUI
}
