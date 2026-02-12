package main

import (
	"bytes"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

func BuildEditorUI(
	assets []AssetInfo,
	prefabs []PrefabInfo,
	onAssetSelected func(asset AssetInfo, setTileset func(img *ebiten.Image)),
	onToolSelected func(tool Tool),
	onTileSelected func(tileIndex int),
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
	initialLayers []string,
	initialLayerIndex int,
	initialTool Tool,
	initialAutotileEnabled bool,
) (*ebitenui.UI, *ToolBar, *LayerPanel, *widget.TextInput, func(img *ebiten.Image), func(tileIndex int), func(enabled bool), *TransitionUI) {
	ui := &ebitenui.UI{}

	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic("Failed to load font: " + err.Error())
	}

	var fontFace text.Face = &text.GoTextFace{Source: s, Size: 14}
	ui.PrimaryTheme = newEditorTheme(&fontFace)

	rightPanel := buildTilesetPanelUI(assets, ui.PrimaryTheme, &fontFace, onAssetSelected, onTileSelected)
	toolbarContainer, toolBar := buildToolBar(ui.PrimaryTheme, &fontFace, onToolSelected, initialTool)
	leftPanel := buildLeftPanelUI(
		ui.PrimaryTheme,
		&fontFace,
		prefabs,
		onLayerSelected,
		onLayerRenamed,
		onNewLayer,
		onMoveLayerUp,
		onMoveLayerDown,
		onTogglePhysics,
		onTogglePhysicsHighlight,
		onToggleAutotile,
		onPrefabSelected,
		onToggleTransitionMode,
		onTransitionFieldChanged,
	)
	gridPanel := buildGridPanel()

	// Root container: anchor layout
	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	leftPanel.Container.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		StretchVertical:    true,
	}
	rightPanel.Container.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		StretchVertical:    true,
	}
	gridPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		StretchVertical:    true,
	}
	// Toolbar: top center
	toolbarContainer.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
	}
	root.AddChild(gridPanel)
	root.AddChild(leftPanel.Container)
	root.AddChild(rightPanel.Container)
	root.AddChild(toolbarContainer)
	if leftPanel.RenameOverlay != nil {
		root.AddChild(leftPanel.RenameOverlay)
		// Ensure modal overlays stretch to cover the root and center their dialogs.
		leftPanel.RenameOverlay.GetWidget().LayoutData = widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			StretchHorizontal:  true,
			StretchVertical:    true,
		}
	}

	ui.Container = root
	if initialLayers != nil {
		leftPanel.LayerPanel.SetLayers(initialLayers)
		leftPanel.LayerPanel.SetSelected(initialLayerIndex)
	}
	leftPanel.LayerPanel.SetAutotileButtonState(initialAutotileEnabled)

	return ui,
		toolBar,
		leftPanel.LayerPanel,
		leftPanel.FileNameInput,
		rightPanel.ApplyTileset,
		rightPanel.SetTilesetSelection,
		rightPanel.SetTilesetSelectionEnabled,
		leftPanel.TransitionUI
}
