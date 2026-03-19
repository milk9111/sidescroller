package editorui

import (
	"image"
	"image/color"
	"strconv"
	"sync"

	"github.com/ebitenui/ebitenui"
	euiimage "github.com/ebitenui/ebitenui/image"
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
	OnToolSelected             func(editorcomponent.ToolKind)
	OnAssetSelected            func(editorio.AssetInfo)
	OnTileSelected             func(model.TileSelection)
	OnSaveTargetChanged        func(string)
	OnSaveRequested            func()
	OnBackgroundColorChanged   func(string)
	OnLayerSelected            func(int)
	OnLayerAdded               func()
	OnLayerMoved               func(int)
	OnLayerRenamed             func(string)
	OnLayerPhysicsToggled      func()
	OnLayerActiveToggled       func()
	OnLayerVisibilityToggled   func()
	OnPhysicsHighlightToggled  func()
	OnAutotileToggled          func()
	OnPrefabSelected           func(editorio.PrefabInfo)
	OnEntitySelected           func(int)
	OnConvertToPrefabConfirmed func(string)
	OnTransitionModeToggled    func()
	OnGateModeToggled          func()
	OnTriggerModeToggled       func()
	OnBreakableWallModeToggled func()
	OnTransitionSelected       func(int)
	OnGateSelected             func(int)
	OnTriggerSelected          func(int)
	OnBreakableWallSelected    func(int)
	OnTransitionEdited         func(editorcomponents.TransitionEditorState)
	OnGateEdited               func(editorcomponents.GateEditorState)
	OnInspectorDocumentSaved   func(string)
	OnLevelResizeRequested     func(string, string)
}

type EditorUI struct {
	UI                   *ebitenui.UI
	Theme                *editorcomponents.Theme
	Toolbar              *editorcomponents.Toolbar
	InfoPanel            *editorcomponents.InfoPanel
	AssetPanel           *editorcomponents.AssetPanel
	convertToPrefabModal *convertToPrefabModal
	resizeLevelModal     *resizeLevelModal
	currentLevelWidth    int
	currentLevelHeight   int
}

type LayoutMetrics struct {
	LeftInset  float64
	RightInset float64
	TopInset   float64
}

func NewEditorUI(assets []editorio.AssetInfo, callbacks Callbacks) (*EditorUI, error) {
	ensureClipboard()
	var editor *EditorUI

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
		{Tool: editorcomponent.ToolBox, Label: "Box"},
		{Tool: editorcomponent.ToolBoxErase, Label: "Box Erase"},
		{Tool: editorcomponent.ToolLine, Label: "Line"},
		{Tool: editorcomponent.ToolMove, Label: "Move"},
		{Tool: editorcomponent.ToolSpike, Label: "Spike"},
	}, callbacks.OnToolSelected, func() {
		if editor != nil {
			editor.openResizeLevelModal()
		}
	})
	toolbar.Root.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchHorizontal:  true,
	}
	toolbar.Root.GetWidget().MinWidth = 0
	toolbar.Root.GetWidget().MinHeight = TopToolbarHeight

	infoPanel := editorcomponents.NewInfoPanel(theme, callbacks.OnSaveTargetChanged, callbacks.OnSaveRequested, callbacks.OnBackgroundColorChanged, editorcomponents.LayerCallbacks{
		OnLayerSelected:           callbacks.OnLayerSelected,
		OnLayerAdded:              callbacks.OnLayerAdded,
		OnLayerMoved:              callbacks.OnLayerMoved,
		OnLayerRenamed:            callbacks.OnLayerRenamed,
		OnLayerPhysicsToggled:     callbacks.OnLayerPhysicsToggled,
		OnLayerActiveToggled:      callbacks.OnLayerActiveToggled,
		OnLayerVisibilityToggled:  callbacks.OnLayerVisibilityToggled,
		OnPhysicsHighlightToggled: callbacks.OnPhysicsHighlightToggled,
		OnAutotileToggled:         callbacks.OnAutotileToggled,
		OnPrefabSelected: func(item editorcomponents.PrefabListItem) {
			if callbacks.OnPrefabSelected != nil {
				callbacks.OnPrefabSelected(editorio.PrefabInfo{Name: item.Name, Path: item.Path, EntityType: item.EntityType})
			}
		},
		OnEntitySelected: callbacks.OnEntitySelected,
		OnConvertToPrefabRequested: func() {
			if editor != nil {
				editor.openConvertToPrefabModal()
			}
		},
		OnTransitionModeToggled:    callbacks.OnTransitionModeToggled,
		OnGateModeToggled:          callbacks.OnGateModeToggled,
		OnTriggerModeToggled:       callbacks.OnTriggerModeToggled,
		OnBreakableWallModeToggled: callbacks.OnBreakableWallModeToggled,
		OnTransitionSelected:       callbacks.OnTransitionSelected,
		OnGateSelected:             callbacks.OnGateSelected,
		OnTriggerSelected:          callbacks.OnTriggerSelected,
		OnBreakableWallSelected:    callbacks.OnBreakableWallSelected,
		OnTransitionEdited:         callbacks.OnTransitionEdited,
		OnGateEdited:               callbacks.OnGateEdited,
	})
	infoPanel.Root.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchVertical:    true,
		Padding:            &widget.Insets{Top: TopToolbarHeight},
	}
	infoPanel.Root.GetWidget().MinWidth = LeftPanelWidth

	assetPanel := editorcomponents.NewAssetPanel(theme, assets, callbacks.OnAssetSelected, callbacks.OnTileSelected, callbacks.OnInspectorDocumentSaved)
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

	modal := newConvertToPrefabModal(theme, func(name string) {
		if callbacks.OnConvertToPrefabConfirmed != nil {
			callbacks.OnConvertToPrefabConfirmed(name)
		}
		if editor != nil {
			editor.closeConvertToPrefabModal()
		}
	}, func() {
		if editor != nil {
			editor.closeConvertToPrefabModal()
		}
	})
	root.AddChild(modal.Root)

	resizeModal := newResizeLevelModal(theme, func(widthText, heightText string) {
		if callbacks.OnLevelResizeRequested != nil {
			callbacks.OnLevelResizeRequested(widthText, heightText)
		}
		if editor != nil {
			editor.closeResizeLevelModal()
		}
	}, func() {
		if editor != nil {
			editor.closeResizeLevelModal()
		}
	})
	root.AddChild(resizeModal.Root)

	editor = &EditorUI{
		UI:                   &ebitenui.UI{Container: root},
		Theme:                theme,
		Toolbar:              toolbar,
		InfoPanel:            infoPanel,
		AssetPanel:           assetPanel,
		convertToPrefabModal: modal,
		resizeLevelModal:     resizeModal,
	}
	return editor, nil
}

func (e *EditorUI) Sync(tool editorcomponent.ToolKind, saveTarget string, width, height, currentLayer, layerCount int, layers []editorcomponents.LayerListItem, autotileEnabled, physicsHighlight, dirty bool, prefabs []editorcomponents.PrefabListItem, selectedPrefabPath string, entities []editorcomponents.EntityListItem, selectedEntity int, transitionMode, gateMode, triggerMode, breakableWallMode bool, transitions, gates, triggers, breakableWalls []editorcomponents.EntityListItem, transitionEditor editorcomponents.TransitionEditorState, gateEditor editorcomponents.GateEditorState, triggerEditor editorcomponents.TriggerEditorState, selectedPath string, selectedIndex int, backgroundColor string, status string, inspector editorcomponents.InspectorState) {
	if e == nil {
		return
	}
	e.currentLevelWidth = width
	e.currentLevelHeight = height
	e.Toolbar.SetActive(tool)
	e.InfoPanel.Sync(editorcomponents.InfoPanelState{
		SaveTarget:         saveTarget,
		Width:              width,
		Height:             height,
		BackgroundColor:    backgroundColor,
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
		TriggerMode:        triggerMode,
		BreakableWallMode:  breakableWallMode,
		Transitions:        transitions,
		Gates:              gates,
		Triggers:           triggers,
		BreakableWalls:     breakableWalls,
		TransitionEditor:   transitionEditor,
		GateEditor:         gateEditor,
		TriggerEditor:      triggerEditor,
		Status:             status,
	})
	e.AssetPanel.Sync(model.TileSelection{Path: selectedPath, Index: selectedIndex}, autotileEnabled, inspector)
}

func (e *EditorUI) Update() {
	if e == nil || e.UI == nil {
		return
	}
	e.UI.Update()
	if e.convertToPrefabModal != nil && e.convertToPrefabModal.Visible() && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		e.closeConvertToPrefabModal()
	}
	if e.resizeLevelModal != nil && e.resizeLevelModal.Visible() && inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		e.closeResizeLevelModal()
	}
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
	if e.FocusedInput() != nil {
		return true
	}
	if e.AssetPanel != nil && e.AssetPanel.Inspector != nil {
		return e.AssetPanel.Inspector.AnyInputFocused()
	}
	return false
}

func (e *EditorUI) CurrentTransitionEditorState() (editorcomponents.TransitionEditorState, bool) {
	if e == nil || e.InfoPanel == nil || e.InfoPanel.TransitionPanel == nil {
		return editorcomponents.TransitionEditorState{}, false
	}
	return e.InfoPanel.TransitionPanel.DraftState()
}

func (e *EditorUI) FocusedInput() *widget.TextInput {
	if e == nil || e.InfoPanel == nil {
		return nil
	}
	if e.InfoPanel.FileInput != nil && e.InfoPanel.FileInput.IsFocused() {
		return e.InfoPanel.FileInput
	}
	if e.InfoPanel.LayerPanel != nil {
		if e.InfoPanel.LayerPanel.SearchInput != nil && e.InfoPanel.LayerPanel.SearchInput.IsFocused() {
			return e.InfoPanel.LayerPanel.SearchInput
		}
		if e.InfoPanel.LayerPanel.RenameInput != nil && e.InfoPanel.LayerPanel.RenameInput.IsFocused() {
			return e.InfoPanel.LayerPanel.RenameInput
		}
	}
	if e.InfoPanel.PrefabPanel != nil && e.InfoPanel.PrefabPanel.SearchInput != nil && e.InfoPanel.PrefabPanel.SearchInput.IsFocused() {
		return e.InfoPanel.PrefabPanel.SearchInput
	}
	if e.InfoPanel.EntityPanel != nil && e.InfoPanel.EntityPanel.SearchInput != nil && e.InfoPanel.EntityPanel.SearchInput.IsFocused() {
		return e.InfoPanel.EntityPanel.SearchInput
	}
	if e.convertToPrefabModal != nil && e.convertToPrefabModal.Input != nil && e.convertToPrefabModal.Input.IsFocused() {
		return e.convertToPrefabModal.Input
	}
	if e.resizeLevelModal != nil {
		if e.resizeLevelModal.WidthInput != nil && e.resizeLevelModal.WidthInput.IsFocused() {
			return e.resizeLevelModal.WidthInput
		}
		if e.resizeLevelModal.HeightInput != nil && e.resizeLevelModal.HeightInput.IsFocused() {
			return e.resizeLevelModal.HeightInput
		}
	}
	if e.InfoPanel.TransitionPanel != nil {
		if e.InfoPanel.TransitionPanel.SearchInput != nil && e.InfoPanel.TransitionPanel.SearchInput.IsFocused() {
			return e.InfoPanel.TransitionPanel.SearchInput
		}
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
	if e.InfoPanel.GatePanel != nil && e.InfoPanel.GatePanel.SearchInput != nil && e.InfoPanel.GatePanel.SearchInput.IsFocused() {
		return e.InfoPanel.GatePanel.SearchInput
	}
	if e.InfoPanel.GatePanel != nil && e.InfoPanel.GatePanel.GroupInput != nil && e.InfoPanel.GatePanel.GroupInput.IsFocused() {
		return e.InfoPanel.GatePanel.GroupInput
	}
	if e.InfoPanel.TriggerPanel != nil && e.InfoPanel.TriggerPanel.SearchInput != nil && e.InfoPanel.TriggerPanel.SearchInput.IsFocused() {
		return e.InfoPanel.TriggerPanel.SearchInput
	}
	if e.InfoPanel.BreakableWallPanel != nil && e.InfoPanel.BreakableWallPanel.SearchInput != nil && e.InfoPanel.BreakableWallPanel.SearchInput.IsFocused() {
		return e.InfoPanel.BreakableWallPanel.SearchInput
	}
	if e.AssetPanel != nil && e.AssetPanel.SearchInput != nil && e.AssetPanel.SearchInput.IsFocused() {
		return e.AssetPanel.SearchInput
	}
	if e.AssetPanel != nil && e.AssetPanel.Inspector != nil {
		if input := e.AssetPanel.Inspector.FocusedInput(); input != nil {
			return input
		}
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

func (e *EditorUI) openConvertToPrefabModal() {
	if e == nil || e.convertToPrefabModal == nil {
		return
	}
	e.convertToPrefabModal.Open()
	e.requestRootRelayout()
}

func (e *EditorUI) closeConvertToPrefabModal() {
	if e == nil || e.convertToPrefabModal == nil {
		return
	}
	e.convertToPrefabModal.Close()
	e.requestRootRelayout()
}

func (e *EditorUI) openResizeLevelModal() {
	if e == nil || e.resizeLevelModal == nil {
		return
	}
	e.resizeLevelModal.Open(e.currentLevelWidth, e.currentLevelHeight)
	e.requestRootRelayout()
}

func (e *EditorUI) closeResizeLevelModal() {
	if e == nil || e.resizeLevelModal == nil {
		return
	}
	e.resizeLevelModal.Close()
	e.requestRootRelayout()
}

func (e *EditorUI) requestRootRelayout() {
	if e == nil || e.UI == nil || e.UI.Container == nil {
		return
	}
	e.UI.Container.RequestRelayout()
}

type convertToPrefabModal struct {
	Root   *widget.Container
	Input  *widget.TextInput
	ok     func(string)
	cancel func()
}

func newConvertToPrefabModal(theme *editorcomponents.Theme, onOK func(string), onCancel func()) *convertToPrefabModal {
	overlay := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{A: 160})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	overlay.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	overlay.GetWidget().Visibility = widget.Visibility_Hide

	dialog := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.PanelBackground),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(16)),
		)),
	)
	dialog.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	dialog.GetWidget().MinWidth = 360

	dialog.AddChild(widget.NewText(
		widget.TextOpts.Text("Convert to Prefab", &theme.TitleFace, theme.TextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	))
	dialog.AddChild(widget.NewText(
		widget.TextOpts.Text("Prefab file name", &theme.Face, theme.MutedTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	))

	modal := &convertToPrefabModal{Root: overlay, ok: onOK, cancel: onCancel}
	modal.Input = widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if modal.ok != nil {
				modal.ok(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	dialog.AddChild(modal.Input)

	buttons := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	buttons.AddChild(widget.NewButton(
		widget.ButtonOpts.Image(theme.ActiveButtonImage),
		widget.ButtonOpts.Text("OK", &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if modal.ok != nil {
				modal.ok(modal.Input.GetText())
			}
		}),
	))
	buttons.AddChild(widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonImage),
		widget.ButtonOpts.Text("Cancel", &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if modal.cancel != nil {
				modal.cancel()
			}
		}),
	))
	dialog.AddChild(buttons)
	overlay.AddChild(dialog)
	return modal
}

func (m *convertToPrefabModal) Open() {
	if m == nil || m.Root == nil {
		return
	}
	m.Root.GetWidget().Visibility = widget.Visibility_Show
	if m.Input != nil {
		m.Input.SetText("")
		m.Input.Focus(true)
	}
}

func (m *convertToPrefabModal) Close() {
	if m == nil || m.Root == nil {
		return
	}
	m.Root.GetWidget().Visibility = widget.Visibility_Hide
	if m.Input != nil {
		m.Input.Focus(false)
	}
}

func (m *convertToPrefabModal) Visible() bool {
	if m == nil || m.Root == nil || m.Root.GetWidget() == nil {
		return false
	}
	return m.Root.GetWidget().Visibility == widget.Visibility_Show
}

type resizeLevelModal struct {
	Root        *widget.Container
	Dialog      *widget.Container
	WidthInput  *widget.TextInput
	HeightInput *widget.TextInput
	ok          func(string, string)
	cancel      func()
}

func newResizeLevelModal(theme *editorcomponents.Theme, onOK func(string, string), onCancel func()) *resizeLevelModal {
	overlay := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{A: 160})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	overlay.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	overlay.GetWidget().Visibility = widget.Visibility_Hide

	dialog := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.PanelBackground),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(widget.NewInsetsSimple(16)),
		)),
	)
	dialog.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	dialog.GetWidget().MinWidth = 360

	dialog.AddChild(widget.NewText(
		widget.TextOpts.Text("Resize Level", &theme.TitleFace, theme.TextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	))
	dialog.AddChild(widget.NewText(
		widget.TextOpts.Text("New width", &theme.Face, theme.MutedTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	))

	modal := &resizeLevelModal{Root: overlay, Dialog: dialog, ok: onOK, cancel: onCancel}
	modal.WidthInput = widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	dialog.AddChild(modal.WidthInput)
	dialog.AddChild(widget.NewText(
		widget.TextOpts.Text("New height", &theme.Face, theme.MutedTextColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	))
	modal.HeightInput = widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.SubmitHandler(func(*widget.TextInputChangedEventArgs) {
			if modal.ok != nil {
				modal.ok(modal.WidthInput.GetText(), modal.HeightInput.GetText())
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	dialog.AddChild(modal.HeightInput)

	buttons := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	buttons.AddChild(widget.NewButton(
		widget.ButtonOpts.Image(theme.ActiveButtonImage),
		widget.ButtonOpts.Text("Apply", &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if modal.ok != nil {
				modal.ok(modal.WidthInput.GetText(), modal.HeightInput.GetText())
			}
		}),
	))
	buttons.AddChild(widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonImage),
		widget.ButtonOpts.Text("Cancel", &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if modal.cancel != nil {
				modal.cancel()
			}
		}),
	))
	dialog.AddChild(buttons)
	overlay.AddChild(dialog)
	return modal
}

func (m *resizeLevelModal) Open(width, height int) {
	if m == nil || m.Root == nil {
		return
	}
	m.Root.GetWidget().Visibility = widget.Visibility_Show
	m.Root.RequestRelayout()
	if m.Dialog != nil {
		m.Dialog.RequestRelayout()
	}
	if m.WidthInput != nil {
		m.WidthInput.SetText(strconv.Itoa(width))
		m.WidthInput.Focus(false)
	}
	if m.HeightInput != nil {
		m.HeightInput.SetText(strconv.Itoa(height))
		m.HeightInput.Focus(true)
		m.HeightInput.SelectAll()
	}
}

func (m *resizeLevelModal) Close() {
	if m == nil || m.Root == nil {
		return
	}
	m.Root.GetWidget().Visibility = widget.Visibility_Hide
	m.Root.RequestRelayout()
	if m.Dialog != nil {
		m.Dialog.RequestRelayout()
	}
	if m.WidthInput != nil {
		m.WidthInput.Focus(false)
	}
	if m.HeightInput != nil {
		m.HeightInput.Focus(false)
	}
}

func (m *resizeLevelModal) Visible() bool {
	if m == nil || m.Root == nil || m.Root.GetWidget() == nil {
		return false
	}
	return m.Root.GetWidget().Visibility == widget.Visibility_Show
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
