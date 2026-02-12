package main

import (
	"fmt"
	"image/color"
	"time"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func addLayersSection(
	parent *widget.Container,
	theme *widget.Theme,
	fontFace *text.Face,
	layerPanel *LayerPanel,
	onLayerSelected func(layerIndex int),
	onTogglePhysics func(),
	onToggleAutotile func(),
	onTogglePhysicsHighlight func(),
) (*widget.List, *widget.Button) {
	layersLabel := widget.NewLabel(
		widget.LabelOpts.Text("Layers", fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	parent.AddChild(layersLabel)

	layerList := widget.NewList(
		widget.ListOpts.Entries([]any{}),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if entry, ok := e.(LayerEntry); ok {
				return fmt.Sprintf("%d. %s", entry.Index+1, entry.Name)
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			entry, ok := args.Entry.(LayerEntry)
			if !ok {
				return
			}
			now := time.Now()
			if layerPanel.suppressEvents {
				layerPanel.lastClickIndex = entry.Index
				layerPanel.lastClickTime = now
				if onLayerSelected != nil {
					onLayerSelected(entry.Index)
				}
				return
			}
			layerPanel.lastClickIndex = entry.Index
			layerPanel.lastClickTime = now
			if onLayerSelected != nil {
				onLayerSelected(entry.Index)
			}
		}),
	)
	parent.AddChild(layerList)
	layerPanel.list = layerList

	buttonsRow := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(6),
			),
		),
	)
	newLayerBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("New", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if layerPanel.onNewLayer != nil {
				layerPanel.onNewLayer()
			}
		}),
	)
	upBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Up", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if layerPanel.onMoveUp == nil {
				return
			}
			if sel, ok := layerList.SelectedEntry().(LayerEntry); ok {
				layerPanel.onMoveUp(sel.Index)
			}
		}),
	)
	downBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Down", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if layerPanel.onMoveDown == nil {
				return
			}
			if sel, ok := layerList.SelectedEntry().(LayerEntry); ok {
				layerPanel.onMoveDown(sel.Index)
			}
		}),
	)
	renameBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Rename", fontFace, theme.ButtonTheme.TextColor),
	)
	buttonsRow.AddChild(newLayerBtn)
	buttonsRow.AddChild(upBtn)
	buttonsRow.AddChild(downBtn)
	buttonsRow.AddChild(renameBtn)
	parent.AddChild(buttonsRow)

	physicsButtonsRow := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(6),
			),
		),
	)
	physicsBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Physics Off", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if onTogglePhysics != nil {
				onTogglePhysics()
			}
		}),
	)
	autotileBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Autotile On", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if onToggleAutotile != nil {
				onToggleAutotile()
			}
		}),
	)
	highlightBtn := widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Highlight Physics", fontFace, theme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if onTogglePhysicsHighlight != nil {
				onTogglePhysicsHighlight()
			}
		}),
	)
	physicsButtonsRow.AddChild(physicsBtn)
	physicsButtonsRow.AddChild(autotileBtn)
	physicsButtonsRow.AddChild(highlightBtn)
	parent.AddChild(physicsButtonsRow)
	layerPanel.physicsBtn = physicsBtn
	layerPanel.autotileBtn = autotileBtn

	return layerList, renameBtn
}
