package components

import (
	"github.com/ebitenui/ebitenui/widget"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
)

type ToolButtonDef struct {
	Tool  editorcomponent.ToolKind
	Label string
}

type Toolbar struct {
	Root    *widget.Container
	buttons map[editorcomponent.ToolKind]*widget.Button
	theme   *Theme
}

func NewToolbar(theme *Theme, defs []ToolButtonDef, onSelected func(editorcomponent.ToolKind), onResizeRequested func()) *Toolbar {
	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.ToolbarBackground),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(12),
			widget.RowLayoutOpts.Padding(theme.PanelPadding),
		)),
	)

	toolbar := &Toolbar{Root: root, buttons: make(map[editorcomponent.ToolKind]*widget.Button), theme: theme}
	for _, def := range defs {
		tool := def.Tool
		button := widget.NewButton(
			widget.ButtonOpts.Image(theme.ButtonImage),
			widget.ButtonOpts.Text(def.Label, &theme.Face, theme.ButtonText),
			widget.ButtonOpts.TextPadding(theme.ButtonPadding),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
				if onSelected != nil {
					onSelected(tool)
				}
			}),
		)
		toolbar.buttons[tool] = button
		root.AddChild(button)
	}
	root.AddChild(widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonImage),
		widget.ButtonOpts.Text("Resize", &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if onResizeRequested != nil {
				onResizeRequested()
			}
		}),
	))
	return toolbar
}

func (t *Toolbar) SetActive(tool editorcomponent.ToolKind) {
	for candidate, button := range t.buttons {
		if candidate == tool {
			button.SetImage(t.theme.ActiveButtonImage)
		} else {
			button.SetImage(t.theme.ButtonImage)
		}
	}
}
