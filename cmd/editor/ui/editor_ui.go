package editorui

import (
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	editorcomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
)

const (
	LeftPanelWidth   = 280
	RightPanelWidth  = 280
	TopToolbarHeight = 56
)

type Callbacks struct {
	OnToolSelected            func(editorcomponent.ToolKind)
	OnAssetSelected           func(editorio.AssetInfo)
	OnTileSelected            func(model.TileSelection)
	OnSaveTargetChanged       func(string)
	OnSaveRequested           func()
	OnLayerSelected           func(int)
	OnLayerAdded              func()
	OnLayerMoved              func(int)
	OnLayerRenamed            func(string)
	OnLayerPhysicsToggled     func()
	OnPhysicsHighlightToggled func()
	OnAutotileToggled         func()
	OnPrefabSelected          func(editorio.PrefabInfo)
	OnEntitySelected          func(int)
	OnTransitionModeToggled   func()
	OnGateModeToggled         func()
	OnTransitionSelected      func(int)
	OnGateSelected            func(int)
	OnTransitionEdited        func(editorcomponents.TransitionEditorState)
	OnGateEdited              func(editorcomponents.GateEditorState)
}

type EditorUI struct {
	UI         *ebitenui.UI
	Theme      *editorcomponents.Theme
	Toolbar    *editorcomponents.Toolbar
	InfoPanel  *editorcomponents.InfoPanel
	AssetPanel *editorcomponents.AssetPanel
}

func NewEditorUI(assets []editorio.AssetInfo, callbacks Callbacks) (*EditorUI, error) {
	theme, err := editorcomponents.NewTheme()
	if err != nil {
		return nil, err
	}

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	toolbar := editorcomponents.NewToolbar(theme, []editorcomponents.ToolButtonDef{
		{Tool: editorcomponent.ToolBrush, Label: "Brush"},
		{Tool: editorcomponent.ToolErase, Label: "Erase"},
		{Tool: editorcomponent.ToolFill, Label: "Fill"},
		{Tool: editorcomponent.ToolLine, Label: "Line"},
		{Tool: editorcomponent.ToolSpike, Label: "Spike"},
	}, callbacks.OnToolSelected)
	toolbar.Root.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchHorizontal:  true,
	}
	toolbar.Root.GetWidget().MinWidth = 0
	toolbar.Root.GetWidget().MinHeight = TopToolbarHeight

	infoPanel := editorcomponents.NewInfoPanel(theme, callbacks.OnSaveTargetChanged, callbacks.OnSaveRequested, editorcomponents.LayerCallbacks{
		OnLayerSelected:           callbacks.OnLayerSelected,
		OnLayerAdded:              callbacks.OnLayerAdded,
		OnLayerMoved:              callbacks.OnLayerMoved,
		OnLayerRenamed:            callbacks.OnLayerRenamed,
		OnLayerPhysicsToggled:     callbacks.OnLayerPhysicsToggled,
		OnPhysicsHighlightToggled: callbacks.OnPhysicsHighlightToggled,
		OnAutotileToggled:         callbacks.OnAutotileToggled,
		OnPrefabSelected: func(item editorcomponents.PrefabListItem) {
			if callbacks.OnPrefabSelected != nil {
				callbacks.OnPrefabSelected(editorio.PrefabInfo{Name: item.Name, Path: item.Path, EntityType: item.EntityType})
			}
		},
		OnEntitySelected:        callbacks.OnEntitySelected,
		OnTransitionModeToggled: callbacks.OnTransitionModeToggled,
		OnGateModeToggled:       callbacks.OnGateModeToggled,
		OnTransitionSelected:    callbacks.OnTransitionSelected,
		OnGateSelected:          callbacks.OnGateSelected,
		OnTransitionEdited:      callbacks.OnTransitionEdited,
		OnGateEdited:            callbacks.OnGateEdited,
	})
	infoPanel.Root.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchVertical:    true,
		Padding:            &widget.Insets{Top: TopToolbarHeight},
	}
	infoPanel.Root.GetWidget().MinWidth = LeftPanelWidth

	assetPanel := editorcomponents.NewAssetPanel(theme, assets, callbacks.OnAssetSelected, callbacks.OnTileSelected)
	assetPanel.Root.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchVertical:    true,
		Padding:            &widget.Insets{Top: TopToolbarHeight},
	}
	assetPanel.Root.GetWidget().MinWidth = RightPanelWidth

	root.AddChild(toolbar.Root)
	root.AddChild(infoPanel.Root)
	root.AddChild(assetPanel.Root)

	return &EditorUI{
		UI:         &ebitenui.UI{Container: root},
		Theme:      theme,
		Toolbar:    toolbar,
		InfoPanel:  infoPanel,
		AssetPanel: assetPanel,
	}, nil
}

func (e *EditorUI) Sync(tool editorcomponent.ToolKind, saveTarget string, width, height, currentLayer, layerCount int, layers []editorcomponents.LayerListItem, autotileEnabled, physicsHighlight, dirty bool, prefabs []editorcomponents.PrefabListItem, selectedPrefabPath string, entities []editorcomponents.EntityListItem, selectedEntity int, transitionMode, gateMode bool, transitions, gates []editorcomponents.EntityListItem, transitionEditor editorcomponents.TransitionEditorState, gateEditor editorcomponents.GateEditorState, selectedPath string, selectedIndex int, status string) {
	if e == nil {
		return
	}
	e.Toolbar.SetActive(tool)
	e.InfoPanel.Sync(editorcomponents.InfoPanelState{
		SaveTarget:         saveTarget,
		Width:              width,
		Height:             height,
		CurrentLayer:       currentLayer,
		LayerCount:         layerCount,
		Layers:             layers,
		Autotile:           autotileEnabled,
		PhysicsHighlight:   physicsHighlight,
		Dirty:              dirty,
		SelectedTile:       model.TileSelection{Path: selectedPath, Index: selectedIndex},
		SelectedPrefabPath: selectedPrefabPath,
		Prefabs:            prefabs,
		Entities:           entities,
		SelectedEntity:     selectedEntity,
		TransitionMode:     transitionMode,
		GateMode:           gateMode,
		Transitions:        transitions,
		Gates:              gates,
		TransitionEditor:   transitionEditor,
		GateEditor:         gateEditor,
		Status:             status,
	})
	e.AssetPanel.Sync(model.TileSelection{Path: selectedPath, Index: selectedIndex}, autotileEnabled)
}

func (e *EditorUI) Update() {
	if e == nil || e.UI == nil {
		return
	}
	e.UI.Update()
}

func (e *EditorUI) Draw(screen *ebiten.Image) {
	if e == nil || e.UI == nil {
		return
	}
	e.UI.Draw(screen)
}

func (e *EditorUI) AnyInputFocused() bool {
	if e == nil || e.InfoPanel == nil {
		return false
	}
	if e.InfoPanel.FileInput != nil && e.InfoPanel.FileInput.IsFocused() {
		return true
	}
	if e.InfoPanel.TransitionPanel != nil {
		if e.InfoPanel.TransitionPanel.IDInput != nil && e.InfoPanel.TransitionPanel.IDInput.IsFocused() {
			return true
		}
		if e.InfoPanel.TransitionPanel.ToLevelInput != nil && e.InfoPanel.TransitionPanel.ToLevelInput.IsFocused() {
			return true
		}
		if e.InfoPanel.TransitionPanel.LinkedInput != nil && e.InfoPanel.TransitionPanel.LinkedInput.IsFocused() {
			return true
		}
	}
	if e.InfoPanel.GatePanel != nil && e.InfoPanel.GatePanel.GroupInput != nil && e.InfoPanel.GatePanel.GroupInput.IsFocused() {
		return true
	}
	return e.InfoPanel.LayerPanel != nil && e.InfoPanel.LayerPanel.RenameInput != nil && e.InfoPanel.LayerPanel.RenameInput.IsFocused()
}
