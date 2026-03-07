package editorui

import (
	"image"
	"sync"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	editorcomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
	"golang.design/x/clipboard"
)

const (
	LeftPanelWidth   = 280
	RightPanelWidth  = 280
	TopToolbarHeight = 56
)

var (
	clipboardInitOnce sync.Once
	clipboardReady    bool
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
	OnLayerVisibilityToggled  func()
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
	OnInspectorFieldEdited    func(editorcomponents.InspectorFieldEdit)
}

type EditorUI struct {
	UI         *ebitenui.UI
	Theme      *editorcomponents.Theme
	Toolbar    *editorcomponents.Toolbar
	InfoPanel  *editorcomponents.InfoPanel
	AssetPanel *editorcomponents.AssetPanel
}

type LayoutMetrics struct {
	LeftInset  float64
	RightInset float64
	TopInset   float64
}

func NewEditorUI(assets []editorio.AssetInfo, callbacks Callbacks) (*EditorUI, error) {
	ensureClipboard()

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
		OnLayerVisibilityToggled:  callbacks.OnLayerVisibilityToggled,
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

	assetPanel := editorcomponents.NewAssetPanel(theme, assets, callbacks.OnAssetSelected, callbacks.OnTileSelected, callbacks.OnInspectorFieldEdited)
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

func (e *EditorUI) Sync(tool editorcomponent.ToolKind, saveTarget string, width, height, currentLayer, layerCount int, layers []editorcomponents.LayerListItem, autotileEnabled, physicsHighlight, dirty bool, prefabs []editorcomponents.PrefabListItem, selectedPrefabPath string, entities []editorcomponents.EntityListItem, selectedEntity int, transitionMode, gateMode bool, transitions, gates []editorcomponents.EntityListItem, transitionEditor editorcomponents.TransitionEditorState, gateEditor editorcomponents.GateEditorState, selectedPath string, selectedIndex int, status string, inspector editorcomponents.InspectorState) {
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
	e.AssetPanel.Sync(model.TileSelection{Path: selectedPath, Index: selectedIndex}, autotileEnabled, inspector)
}

func (e *EditorUI) Update() {
	if e == nil || e.UI == nil {
		return
	}
	e.UI.Update()
	e.handleFocusedInputShortcuts()
	if e.InfoPanel != nil {
		e.InfoPanel.SuppressAutoListScroll()
	}
	if e.AssetPanel != nil {
		e.AssetPanel.SuppressAutoListScroll()
	}
}

func (e *EditorUI) Draw(screen *ebiten.Image) {
	if e == nil || e.UI == nil {
		return
	}
	e.UI.Draw(screen)
}

func (e *EditorUI) PointerOverUI(x, y int) bool {
	if e == nil || e.UI == nil || e.UI.Container == nil {
		return false
	}
	return widgetBlocksCanvasAt(e.UI.Container, image.Pt(x, y))
}

func (e *EditorUI) LayoutMetrics(screenWidth, screenHeight int) LayoutMetrics {
	metrics := LayoutMetrics{
		LeftInset:  LeftPanelWidth,
		RightInset: RightPanelWidth,
		TopInset:   TopToolbarHeight,
	}
	if e == nil {
		return metrics
	}
	if e.InfoPanel != nil && e.InfoPanel.Root != nil && e.InfoPanel.Root.GetWidget() != nil {
		rect := e.InfoPanel.Root.GetWidget().Rect
		if rect.Dx() > 0 {
			metrics.LeftInset = float64(rect.Max.X)
		}
	}
	if e.AssetPanel != nil && e.AssetPanel.Root != nil && e.AssetPanel.Root.GetWidget() != nil {
		rect := e.AssetPanel.Root.GetWidget().Rect
		if rect.Dx() > 0 {
			metrics.RightInset = float64(maxInt(0, screenWidth-rect.Min.X))
		}
	}
	if e.Toolbar != nil && e.Toolbar.Root != nil && e.Toolbar.Root.GetWidget() != nil {
		rect := e.Toolbar.Root.GetWidget().Rect
		if rect.Dy() > 0 {
			metrics.TopInset = float64(rect.Max.Y)
		}
	}
	if screenHeight <= 0 {
		return metrics
	}
	if metrics.TopInset > float64(screenHeight) {
		metrics.TopInset = float64(screenHeight)
	}
	return metrics
}

func (e *EditorUI) AnyInputFocused() bool {
	return e.FocusedInput() != nil
}

func (e *EditorUI) FocusedInput() *widget.TextInput {
	if e == nil || e.InfoPanel == nil {
		return nil
	}
	if e.InfoPanel.FileInput != nil && e.InfoPanel.FileInput.IsFocused() {
		return e.InfoPanel.FileInput
	}
	if e.InfoPanel.TransitionPanel != nil {
		if e.InfoPanel.TransitionPanel.IDInput != nil && e.InfoPanel.TransitionPanel.IDInput.IsFocused() {
			return e.InfoPanel.TransitionPanel.IDInput
		}
		if e.InfoPanel.TransitionPanel.ToLevelInput != nil && e.InfoPanel.TransitionPanel.ToLevelInput.IsFocused() {
			return e.InfoPanel.TransitionPanel.ToLevelInput
		}
		if e.InfoPanel.TransitionPanel.LinkedInput != nil && e.InfoPanel.TransitionPanel.LinkedInput.IsFocused() {
			return e.InfoPanel.TransitionPanel.LinkedInput
		}
	}
	if e.InfoPanel.GatePanel != nil && e.InfoPanel.GatePanel.GroupInput != nil && e.InfoPanel.GatePanel.GroupInput.IsFocused() {
		return e.InfoPanel.GatePanel.GroupInput
	}
	if e.AssetPanel != nil && e.AssetPanel.Inspector != nil {
		if input := e.AssetPanel.Inspector.FocusedInput(); input != nil {
			return input
		}
	}
	if e.InfoPanel.LayerPanel != nil && e.InfoPanel.LayerPanel.RenameInput != nil && e.InfoPanel.LayerPanel.RenameInput.IsFocused() {
		return e.InfoPanel.LayerPanel.RenameInput
	}
	return nil
}

func (e *EditorUI) handleFocusedInputShortcuts() {
	input := e.FocusedInput()
	if input == nil || !modifierPressed() {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		input.SelectAll()
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		if !ensureClipboard() {
			return
		}
		text := input.SelectedText()
		if text != "" {
			clipboard.Write(clipboard.FmtText, []byte(text))
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyV) {
		if !ensureClipboard() {
			return
		}
		text := string(clipboard.Read(clipboard.FmtText))
		if text != "" {
			input.Insert(text)
		}
		return
	}
}

func ensureClipboard() bool {
	clipboardInitOnce.Do(func() {
		clipboardReady = clipboard.Init() == nil
	})
	return clipboardReady
}

func modifierPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyControlRight) ||
		ebiten.IsKeyPressed(ebiten.KeyMetaLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyMetaRight)
}

func widgetBlocksCanvasAt(node widget.PreferredSizeLocateableWidget, point image.Point) bool {
	if node == nil {
		return false
	}
	state := node.GetWidget()
	if state == nil {
		return false
	}
	if state.Visibility == widget.Visibility_Hide || !point.In(state.Rect) {
		return false
	}
	if container, ok := node.(widget.Containerer); ok {
		children := container.Children()
		for index := len(children) - 1; index >= 0; index-- {
			if widgetBlocksCanvasAt(children[index], point) {
				return true
			}
		}
	}
	return state.TrackHover
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
