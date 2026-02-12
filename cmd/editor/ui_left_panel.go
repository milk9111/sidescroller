package main

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func buildLeftPanelUI(
	theme *widget.Theme,
	fontFace *text.Face,
	prefabs []PrefabInfo,
	onLayerSelected func(layerIndex int),
	onLayerRenamed func(layerIndex int, newName string),
	onNewLayer func(),
	onMoveLayerUp func(layerIndex int),
	onMoveLayerDown func(layerIndex int),
	onTogglePhysics func(),
	onTogglePhysicsHighlight func(),
	onToggleAutotile func(),
	onPrefabSelected func(prefab PrefabInfo),
	onToggleTransitionMode func(enabled bool),
	onTransitionFieldChanged func(field, value string),
) *LeftPanelUI {
	layerPanel := NewLayerPanel()
	layerPanel.onNewLayer = onNewLayer
	layerPanel.onMoveUp = onMoveLayerUp
	layerPanel.onMoveDown = onMoveLayerDown

	leftPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(200, 400),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{40, 40, 40, 255})),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)

	fileNameInput := addFileNameSection(leftPanel, fontFace)

	layerList, renameBtn := addLayersSection(
		leftPanel,
		theme,
		fontFace,
		layerPanel,
		onLayerSelected,
		onTogglePhysics,
		onToggleAutotile,
		onTogglePhysicsHighlight,
	)

	addPrefabsSection(leftPanel, fontFace, prefabs, onPrefabSelected)

	transitionUI := newTransitionSection(leftPanel, theme, fontFace, onToggleTransitionMode, onTransitionFieldChanged)

	renameDialog := newLayerRenameDialog(theme, fontFace, onLayerRenamed)
	layerPanel.openRenameDialog = renameDialog.Open

	// Wire the Rename button to open the rename dialog for the currently selected layer.
	if renameBtn != nil {
		renameBtn.ClickedEvent.AddHandler(event.WrapHandler(func(args *widget.ButtonClickedEventArgs) {
			se := layerList.SelectedEntry()
			if se == nil {
				return
			}
			if sel, ok := se.(LayerEntry); ok {
				name := sel.Name
				if name == "" {
					name = fmt.Sprintf("Layer %d", sel.Index)
				}
				if layerPanel.openRenameDialog != nil {
					layerPanel.openRenameDialog(sel.Index, name)
				}
			}
		}))
	}

	return &LeftPanelUI{
		Container:     leftPanel,
		LayerPanel:    layerPanel,
		FileNameInput: fileNameInput,
		RenameOverlay: renameDialog.Overlay,
		TransitionUI:  transitionUI,
	}
}
