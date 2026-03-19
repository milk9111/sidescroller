package components

import (
	"fmt"
	"image/color"
	"reflect"
	"strings"
	"unsafe"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
)

type InfoPanelState struct {
	SaveTarget         string
	Width              int
	Height             int
	CurrentLayer       int
	LayerCount         int
	Layers             []LayerListItem
	Autotile           bool
	PhysicsHighlight   bool
	Dirty              bool
	SelectedTile       model.TileSelection
	SelectedPrefabPath string
	Prefabs            []PrefabListItem
	Entities           []EntityListItem
	SelectedEntity     int
	TransitionMode     bool
	GateMode           bool
	TriggerMode        bool
	BreakableWallMode  bool
	Transitions        []EntityListItem
	Gates              []EntityListItem
	Triggers           []EntityListItem
	BreakableWalls     []EntityListItem
	TransitionEditor   TransitionEditorState
	GateEditor         GateEditorState
	TriggerEditor      TriggerEditorState
	Status             string
	BackgroundColor    string
}

type TransitionEditorState struct {
	Selected bool
	ID       string
	ToLevel  string
	LinkedID string
	EnterDir string
}

type GateEditorState struct {
	Selected bool
	Group    string
}

type TriggerEditorState struct {
	Selected bool
}

type LayerListItem struct {
	Index   int
	Name    string
	Physics bool
	Active  bool
	Visible bool
}

type PrefabListItem struct {
	Path       string
	Name       string
	EntityType string
}

type EntityListItem struct {
	Index int
	Label string
}

type LayerCallbacks struct {
	OnLayerSelected            func(int)
	OnLayerAdded               func()
	OnLayerMoved               func(int)
	OnLayerRenamed             func(string)
	OnLayerPhysicsToggled      func()
	OnLayerActiveToggled       func()
	OnLayerVisibilityToggled   func()
	OnPhysicsHighlightToggled  func()
	OnAutotileToggled          func()
	OnPrefabSelected           func(PrefabListItem)
	OnEntitySelected           func(int)
	OnConvertToPrefabRequested func()
	OnTransitionModeToggled    func()
	OnGateModeToggled          func()
	OnTriggerModeToggled       func()
	OnBreakableWallModeToggled func()
	OnTransitionSelected       func(int)
	OnGateSelected             func(int)
	OnTriggerSelected          func(int)
	OnBreakableWallSelected    func(int)
	OnTransitionEdited         func(TransitionEditorState)
	OnGateEdited               func(GateEditorState)
}

type InfoPanel struct {
	Root                 *widget.Container
	Scroll               *widget.ScrollContainer
	content              *widget.Container
	FileInput            *widget.TextInput
	BackgroundColorInput *widget.TextInput
	SizeText             *widget.Text
	LayerText            *widget.Text
	DirtyText            *widget.Text
	SelectedText         *widget.Text
	StatusText           *widget.Text
	LayerPanel           *LayerPanel
	PrefabPanel          *PrefabPanel
	EntityPanel          *EntityPanel
	TransitionPanel      *TransitionPanel
	GatePanel            *GatePanel
	TriggerPanel         *TriggerPanel
	BreakableWallPanel   *BreakableWallPanel
}

func NewInfoPanel(theme *Theme, onSaveTargetChanged func(string), onSaveRequested func(), onBackgroundColorChanged func(string), layerCallbacks LayerCallbacks) *InfoPanel {
	root, content, scroll := newScrollablePanel(theme, 10)

	panel := &InfoPanel{Root: root, Scroll: scroll, content: content}
	content.AddChild(newSectionTitle("File", theme))
	panel.FileInput = widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			if onSaveTargetChanged != nil {
				onSaveTargetChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if onSaveTargetChanged != nil {
				onSaveTargetChanged(args.InputText)
			}
			if onSaveRequested != nil {
				onSaveRequested()
			}
		}),
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(panelTextLayoutData()),
		),
	)
	content.AddChild(panel.FileInput)
	content.AddChild(newActionButton(theme, "Save", onSaveRequested))

	content.AddChild(newSectionTitle("Level", theme))
	panel.SizeText = newValueText(theme)
	panel.LayerText = newValueText(theme)
	panel.DirtyText = newValueText(theme)
	content.AddChild(panel.SizeText)
	content.AddChild(panel.LayerText)
	content.AddChild(panel.DirtyText)

	// Background color input
	bgLabel := newValueText(theme)
	bgLabel.Label = "Background color"
	panel.BackgroundColorInput = widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Placeholder("#RRGGBB or #RRGGBBAA"),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			if onBackgroundColorChanged != nil {
				onBackgroundColorChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if onBackgroundColorChanged != nil {
				onBackgroundColorChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(panelTextLayoutData()),
		),
	)
	content.AddChild(bgLabel)
	content.AddChild(panel.BackgroundColorInput)

	panel.LayerPanel = NewLayerPanel(theme, layerCallbacks)
	content.AddChild(panel.LayerPanel.Root)

	panel.PrefabPanel = NewPrefabPanel(theme, layerCallbacks.OnPrefabSelected)
	content.AddChild(panel.PrefabPanel.Root)

	panel.EntityPanel = NewEntityPanel(theme, layerCallbacks.OnEntitySelected)
	content.AddChild(panel.EntityPanel.Root)
	content.AddChild(newActionButton(theme, "Convert to Prefab", layerCallbacks.OnConvertToPrefabRequested))

	panel.TransitionPanel = NewTransitionPanel(theme, layerCallbacks)
	content.AddChild(panel.TransitionPanel.Root)

	panel.GatePanel = NewGatePanel(theme, layerCallbacks)
	content.AddChild(panel.GatePanel.Root)

	panel.TriggerPanel = NewTriggerPanel(theme, layerCallbacks)
	content.AddChild(panel.TriggerPanel.Root)

	panel.BreakableWallPanel = NewBreakableWallPanel(theme, layerCallbacks)
	content.AddChild(panel.BreakableWallPanel.Root)

	content.AddChild(newSectionTitle("Selection", theme))
	panel.SelectedText = newValueText(theme)
	content.AddChild(panel.SelectedText)

	content.AddChild(newSectionTitle("Status", theme))
	panel.StatusText = newValueText(theme)
	content.AddChild(panel.StatusText)

	return panel
}

func (p *InfoPanel) Sync(state InfoPanelState) {
	if p.FileInput != nil && !p.FileInput.IsFocused() && p.FileInput.GetText() != state.SaveTarget {
		p.FileInput.SetText(state.SaveTarget)
	}
	if p.BackgroundColorInput != nil && !p.BackgroundColorInput.IsFocused() && p.BackgroundColorInput.GetText() != state.BackgroundColor {
		p.BackgroundColorInput.SetText(state.BackgroundColor)
	}
	if p.SizeText != nil {
		p.SizeText.Label = fmt.Sprintf("Size: %dx%d", state.Width, state.Height)
	}
	if p.LayerText != nil {
		p.LayerText.Label = fmt.Sprintf("Layer: %d/%d", state.CurrentLayer+1, max(1, state.LayerCount))
	}
	if p.DirtyText != nil {
		p.DirtyText.Label = fmt.Sprintf("Dirty: %t", state.Dirty)
	}
	if p.LayerPanel != nil {
		p.LayerPanel.Sync(state.Layers, state.CurrentLayer, state.Autotile, state.PhysicsHighlight)
	}
	if p.PrefabPanel != nil {
		p.PrefabPanel.Sync(state.Prefabs, state.SelectedPrefabPath)
	}
	if p.EntityPanel != nil {
		p.EntityPanel.Sync(state.Entities, state.SelectedEntity)
	}
	if p.TransitionPanel != nil {
		p.TransitionPanel.Sync(state.TransitionMode, state.Transitions, state.SelectedEntity, state.TransitionEditor)
	}
	if p.GatePanel != nil {
		p.GatePanel.Sync(state.GateMode, state.Gates, state.SelectedEntity, state.GateEditor)
	}
	if p.TriggerPanel != nil {
		p.TriggerPanel.Sync(state.TriggerMode, state.Triggers, state.SelectedEntity, state.TriggerEditor)
	}
	if p.BreakableWallPanel != nil {
		p.BreakableWallPanel.Sync(state.BreakableWallMode, state.BreakableWalls, state.SelectedEntity)
	}
	if p.SelectedText != nil {
		selectedPrefab := state.SelectedPrefabPath
		if selectedPrefab == "" {
			selectedPrefab = "—"
		}
		selectedEntity := "—"
		if state.SelectedEntity >= 0 && state.SelectedEntity < len(state.Entities) {
			selectedEntity = state.Entities[state.SelectedEntity].Label
		}
		p.SelectedText.Label = fmt.Sprintf("Tile: %s #%d\nPrefab: %s\nEntity: %s", state.SelectedTile.Path, state.SelectedTile.Index, selectedPrefab, selectedEntity)
	}
	if p.StatusText != nil {
		p.StatusText.Label = state.Status
	}
}

type AssetPanel struct {
	Root         *widget.Container
	Scroll       *widget.ScrollContainer
	content      *widget.Container
	assetContent *widget.Container
	SelectedText *widget.Text
	SearchInput  *widget.TextInput
	list         *widget.List
	searchList   *SearchableList
	Tileset      *TilesetPicker
	Inspector    *InspectorPanel
	assets       []editorio.AssetInfo
	entries      []any
	interactive  bool
	syncing      bool
}

func NewAssetPanel(theme *Theme, assets []editorio.AssetInfo, onSelected func(editorio.AssetInfo), onTileSelected func(model.TileSelection), onInspectorDocumentSaved func(string)) *AssetPanel {
	root, content, scroll := newScrollablePanel(theme, 8)
	filteredAssets := filterTileAssets(assets)
	panel := &AssetPanel{Root: root, Scroll: scroll, content: content, SelectedText: newValueText(theme), assets: filteredAssets, interactive: true}
	panel.assetContent = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	content.AddChild(panel.assetContent)
	panel.assetContent.AddChild(newSectionTitle("Assets", theme))
	panel.assetContent.AddChild(panel.SelectedText)
	panel.entries = make([]any, 0, len(filteredAssets))
	for _, asset := range filteredAssets {
		panel.entries = append(panel.entries, asset)
	}
	panel.searchList = NewSearchableList(theme, panel.entries, func(entry any) string {
		asset, _ := entry.(editorio.AssetInfo)
		if asset.Relative != "" && asset.Relative != asset.Name {
			return fmt.Sprintf("%s · %s", asset.Name, asset.Relative)
		}
		return asset.Name
	}, func(entry any) {
		if panel.syncing || !panel.interactive || onSelected == nil {
			return
		}
		asset, ok := entry.(editorio.AssetInfo)
		if ok {
			onSelected(asset)
		}
	})
	panel.list = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.list, 220)
	panel.assetContent.AddChild(panel.searchList.Root)
	panel.Tileset = NewTilesetPicker(theme, onTileSelected)
	panel.assetContent.AddChild(panel.Tileset.Root)
	panel.Inspector = NewInspectorPanel(theme, onInspectorDocumentSaved)
	setWidgetVisible(panel.Inspector.Root, false)
	content.AddChild(panel.Inspector.Root)
	return panel
}

func (p *AssetPanel) Sync(selection model.TileSelection, autotileEnabled bool, inspector InspectorState) {
	showInspector := inspector.Active
	p.setInteractive(!showInspector)
	setWidgetVisible(p.assetContent, !showInspector)
	if p.searchList != nil {
		p.searchList.SetVisible(!showInspector)
	}
	if p.Tileset != nil {
		setWidgetVisible(p.Tileset.Root, !showInspector)
	}
	if p.Inspector != nil {
		setWidgetVisible(p.Inspector.Root, showInspector)
		p.Inspector.Sync(inspector)
		if showInspector && p.Scroll != nil {
			if p.Inspector.SetAvailableHeight(p.Scroll.ViewRect().Dy()) {
				p.Root.RequestRelayout()
			}
		}
	}
	if showInspector {
		return
	}
	if p.SelectedText != nil {
		p.SelectedText.Label = fmt.Sprintf("Selected: %s #%d", selection.Path, selection.Index)
	}
	if p.list == nil {
		return
	}
	if p.Scroll != nil {
		panelHeight := p.Scroll.ViewRect().Dy()
		if panelHeight > 0 {
			maxVisibleHeight := panelHeight / 2
			if maxVisibleHeight < 1 {
				maxVisibleHeight = 1
			}
			if applyListHeight(p.list, maxVisibleHeight) {
				p.Root.RequestRelayout()
			}
		}
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	var selectedAsset *editorio.AssetInfo
	for _, entry := range p.entries {
		asset, _ := entry.(editorio.AssetInfo)
		if asset.Name == selection.Path || asset.Relative == selection.Path {
			selectedAsset = &asset
			p.searchList.SetSelectedEntry(entry)
			break
		}
	}
	if selectedAsset == nil {
		p.searchList.SetSelectedEntry(nil)
	}
	if p.Tileset != nil {
		p.Tileset.Sync(selectedAsset, selection, !autotileEnabled)
	}
}

func (p *AssetPanel) setInteractive(enabled bool) {
	if p == nil {
		return
	}
	p.interactive = enabled
	if p.searchList != nil {
		p.searchList.SetEnabled(enabled)
	}
	if p.list != nil {
		p.list.GetWidget().Disabled = !enabled
	}
	if p.Tileset != nil {
		p.Tileset.SetInteractive(enabled)
	}
}

func filterTileAssets(assets []editorio.AssetInfo) []editorio.AssetInfo {
	filtered := make([]editorio.AssetInfo, 0, len(assets))
	for _, asset := range assets {
		if assetContainsTile(asset) {
			filtered = append(filtered, asset)
		}
	}
	return filtered
}

func assetContainsTile(asset editorio.AssetInfo) bool {
	for _, value := range []string{asset.Name, asset.Relative, asset.DiskPath} {
		if strings.Contains(strings.ToLower(value), "tile") {
			return true
		}
	}
	return false
}

func (p *AssetPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.list)
}

type LayerPanel struct {
	Root             *widget.Container
	SearchInput      *widget.TextInput
	List             *widget.List
	RenameInput      *widget.TextInput
	PhysicsButton    *widget.Button
	ActiveButton     *widget.Button
	VisibilityButton *widget.Button
	HighlightButton  *widget.Button
	AutotileButton   *widget.Button
	searchList       *SearchableList
	entries          []any
	syncing          bool
}

func NewLayerPanel(theme *Theme, callbacks LayerCallbacks) *LayerPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &LayerPanel{Root: root}
	root.AddChild(newSectionTitle("Layers", theme))
	panel.searchList = NewSearchableList(theme, nil, func(entry any) string {
		item, _ := entry.(LayerListItem)
		tags := make([]string, 0, 3)
		if item.Physics {
			tags = append(tags, "P")
		}
		if !item.Active {
			tags = append(tags, "Inactive")
		}
		if !item.Visible {
			tags = append(tags, "Hidden")
		}
		if len(tags) > 0 {
			return fmt.Sprintf("%d. %s [%s]", item.Index+1, item.Name, strings.Join(tags, ", "))
		}
		return fmt.Sprintf("%d. %s", item.Index+1, item.Name)
	}, func(entry any) {
		if panel.syncing || callbacks.OnLayerSelected == nil {
			return
		}
		item, ok := entry.(LayerListItem)
		if ok {
			callbacks.OnLayerSelected(item.Index)
		}
	})
	panel.List = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.List, 180)
	root.AddChild(panel.searchList.Root)

	actionsRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	actionsRow.AddChild(newCompactButton(theme, "New", callbacks.OnLayerAdded))
	actionsRow.AddChild(newCompactButton(theme, "Up", func() {
		if callbacks.OnLayerMoved != nil {
			callbacks.OnLayerMoved(-1)
		}
	}))
	actionsRow.AddChild(newCompactButton(theme, "Down", func() {
		if callbacks.OnLayerMoved != nil {
			callbacks.OnLayerMoved(1)
		}
	}))
	root.AddChild(actionsRow)

	panel.RenameInput = widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if callbacks.OnLayerRenamed != nil {
				callbacks.OnLayerRenamed(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(panelTextLayoutData())),
	)
	root.AddChild(panel.RenameInput)
	root.AddChild(newActionButton(theme, "Rename", func() {
		if callbacks.OnLayerRenamed != nil && panel.RenameInput != nil {
			callbacks.OnLayerRenamed(panel.RenameInput.GetText())
		}
	}))
	panel.PhysicsButton = newActionButton(theme, "Physics: Off", callbacks.OnLayerPhysicsToggled)
	panel.ActiveButton = newActionButton(theme, "Active: On", callbacks.OnLayerActiveToggled)
	panel.VisibilityButton = newActionButton(theme, "Visible: On", callbacks.OnLayerVisibilityToggled)
	panel.HighlightButton = newActionButton(theme, "Highlight: Off", callbacks.OnPhysicsHighlightToggled)
	panel.AutotileButton = newActionButton(theme, "Autotile: Off", callbacks.OnAutotileToggled)
	root.AddChild(panel.PhysicsButton)
	root.AddChild(panel.ActiveButton)
	root.AddChild(panel.VisibilityButton)
	root.AddChild(panel.HighlightButton)
	root.AddChild(panel.AutotileButton)
	return panel
}

func (p *LayerPanel) Sync(items []LayerListItem, currentLayer int, autotileEnabled, physicsHighlight bool) {
	if p == nil {
		return
	}
	nextEntries := make([]any, 0, len(items))
	for _, item := range items {
		nextEntries = append(nextEntries, item)
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	if p.List != nil {
		if !entriesEqual(p.entries, nextEntries) {
			p.entries = nextEntries
			p.searchList.SetEntries(p.entries)
		} else {
			p.entries = nextEntries
		}
		if currentLayer >= 0 && currentLayer < len(p.entries) {
			selected := p.entries[currentLayer]
			if p.List.SelectedEntry() != selected {
				p.searchList.SetSelectedEntry(selected)
			}
		}
	}
	if currentLayer >= 0 && currentLayer < len(items) {
		if p.RenameInput != nil && !p.RenameInput.IsFocused() && p.RenameInput.GetText() != items[currentLayer].Name {
			p.RenameInput.SetText(items[currentLayer].Name)
		}
		if p.PhysicsButton != nil {
			label := "Physics: Off"
			if items[currentLayer].Physics {
				label = "Physics: On"
			}
			p.PhysicsButton.SetText(label)
		}
		if p.ActiveButton != nil {
			label := "Active: Off"
			if items[currentLayer].Active {
				label = "Active: On"
			}
			p.ActiveButton.SetText(label)
		}
		if p.VisibilityButton != nil {
			label := "Visible: Off"
			if items[currentLayer].Visible {
				label = "Visible: On"
			}
			p.VisibilityButton.SetText(label)
		}
	}
	if p.HighlightButton != nil {
		label := "Highlight: Off"
		if physicsHighlight {
			label = "Highlight: On"
		}
		p.HighlightButton.SetText(label)
	}
	if p.AutotileButton != nil {
		label := "Autotile: Off"
		if autotileEnabled {
			label = "Autotile: On"
		}
		p.AutotileButton.SetText(label)
	}
}

func (p *LayerPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.List)
}

type PrefabPanel struct {
	Root        *widget.Container
	SearchInput *widget.TextInput
	List        *widget.List
	searchList  *SearchableList
	entries     []any
	syncing     bool
}

func NewPrefabPanel(theme *Theme, onSelected func(PrefabListItem)) *PrefabPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &PrefabPanel{Root: root}
	root.AddChild(newSectionTitle("Prefabs", theme))
	panel.searchList = NewSearchableList(theme, nil, func(entry any) string {
		item, _ := entry.(PrefabListItem)
		return item.Name
	}, func(entry any) {
		if panel.syncing || onSelected == nil {
			return
		}
		item, ok := entry.(PrefabListItem)
		if ok {
			onSelected(item)
		}
	})
	panel.List = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.List, 220)
	root.AddChild(panel.searchList.Root)
	return panel
}

func (p *PrefabPanel) Sync(items []PrefabListItem, selectedPath string) {
	if p == nil || p.List == nil {
		return
	}
	nextEntries := make([]any, 0, len(items))
	for _, item := range items {
		nextEntries = append(nextEntries, item)
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	if !entriesEqual(p.entries, nextEntries) {
		p.entries = nextEntries
		p.searchList.SetEntries(p.entries)
	} else {
		p.entries = nextEntries
	}
	for _, entry := range p.entries {
		item, _ := entry.(PrefabListItem)
		if item.Path == selectedPath {
			p.searchList.SetSelectedEntry(entry)
			return
		}
	}
	if selectedPath == "" {
		p.searchList.SetSelectedEntry(nil)
	}
}

func (p *PrefabPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.List)
}

type EntityPanel struct {
	Root        *widget.Container
	SearchInput *widget.TextInput
	List        *widget.List
	searchList  *SearchableList
	entries     []any
	syncing     bool
}

func NewEntityPanel(theme *Theme, onSelected func(int)) *EntityPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &EntityPanel{Root: root}
	root.AddChild(newSectionTitle("Active Entities", theme))
	panel.searchList = NewSearchableList(theme, nil, func(entry any) string {
		item, _ := entry.(EntityListItem)
		return item.Label
	}, func(entry any) {
		if panel.syncing || onSelected == nil {
			return
		}
		item, ok := entry.(EntityListItem)
		if ok {
			onSelected(item.Index)
		}
	})
	panel.List = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.List, 140)
	root.AddChild(panel.searchList.Root)
	return panel
}

func (p *EntityPanel) Sync(items []EntityListItem, selectedIndex int) {
	if p == nil || p.List == nil {
		return
	}
	nextEntries := make([]any, 0, len(items))
	for _, item := range items {
		nextEntries = append(nextEntries, item)
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	if !entriesEqual(p.entries, nextEntries) {
		p.entries = nextEntries
		p.searchList.SetEntries(p.entries)
	} else {
		p.entries = nextEntries
	}
	for _, entry := range p.entries {
		item, _ := entry.(EntityListItem)
		if item.Index == selectedIndex {
			p.searchList.SetSelectedEntry(entry)
			return
		}
	}
	if selectedIndex < 0 {
		p.searchList.SetSelectedEntry(nil)
	}
}

func (p *EntityPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.List)
}

type TransitionPanel struct {
	Root          *widget.Container
	SearchInput   *widget.TextInput
	List          *widget.List
	ModeButton    *widget.Button
	IDInput       *widget.TextInput
	ToLevelInput  *widget.TextInput
	LinkedInput   *widget.TextInput
	DirButtons    map[string]*widget.Button
	searchList    *SearchableList
	entries       []any
	syncing       bool
	callbacks     LayerCallbacks
	currentState  TransitionEditorState
	selectedIndex int
	draftDirty    bool
	theme         *Theme
}

func NewTransitionPanel(theme *Theme, callbacks LayerCallbacks) *TransitionPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &TransitionPanel{Root: root, callbacks: callbacks, DirButtons: make(map[string]*widget.Button), selectedIndex: -1, theme: theme}
	root.AddChild(newSectionTitle("Transitions", theme))
	panel.ModeButton = newActionButton(theme, "Transitions: Off", callbacks.OnTransitionModeToggled)
	root.AddChild(panel.ModeButton)
	panel.searchList = NewSearchableList(theme, nil, func(entry any) string {
		item, _ := entry.(EntityListItem)
		return item.Label
	}, func(entry any) {
		if panel.syncing || callbacks.OnTransitionSelected == nil {
			return
		}
		item, ok := entry.(EntityListItem)
		if ok {
			callbacks.OnTransitionSelected(item.Index)
		}
	})
	panel.List = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.List, 120)
	root.AddChild(panel.searchList.Root)
	panel.IDInput = newEditorTextInput(theme, func(value string) {
		panel.currentState.ID = value
		panel.currentState.Selected = true
		panel.draftDirty = true
		panel.emitEdit()
	})
	panel.ToLevelInput = newEditorTextInput(theme, func(value string) {
		panel.currentState.ToLevel = value
		panel.currentState.Selected = true
		panel.draftDirty = true
		panel.emitEdit()
	})
	panel.LinkedInput = newEditorTextInput(theme, func(value string) {
		panel.currentState.LinkedID = value
		panel.currentState.Selected = true
		panel.draftDirty = true
		panel.emitEdit()
	})
	root.AddChild(newValueText(theme))
	root.AddChild(widget.NewText(widget.TextOpts.Text("ID", &theme.Face, theme.MutedTextColor)))
	root.AddChild(panel.IDInput)
	root.AddChild(widget.NewText(widget.TextOpts.Text("To Level", &theme.Face, theme.MutedTextColor)))
	root.AddChild(panel.ToLevelInput)
	root.AddChild(widget.NewText(widget.TextOpts.Text("Linked ID", &theme.Face, theme.MutedTextColor)))
	root.AddChild(panel.LinkedInput)
	dirRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(6),
		)),
	)
	for _, direction := range []string{"up", "down", "left", "right"} {
		dir := direction
		button := newCompactButton(theme, strings.ToUpper(dir[:1])+dir[1:], func() {
			panel.currentState.EnterDir = dir
			panel.currentState.Selected = true
			panel.draftDirty = true
			panel.syncDirButtons()
			panel.emitEdit()
		})
		panel.DirButtons[dir] = button
		dirRow.AddChild(button)
	}
	root.AddChild(widget.NewText(widget.TextOpts.Text("Enter Direction", &theme.Face, theme.MutedTextColor)))
	root.AddChild(dirRow)
	return panel
}

func (p *TransitionPanel) Sync(active bool, items []EntityListItem, selectedIndex int, state TransitionEditorState) {
	if p == nil {
		return
	}
	label := "Transitions: Off"
	if active {
		label = "Transitions: On"
	}
	p.ModeButton.SetText(label)
	nextEntries := make([]any, 0, len(items))
	for _, item := range items {
		nextEntries = append(nextEntries, item)
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	if !entriesEqual(p.entries, nextEntries) {
		p.entries = nextEntries
		p.searchList.SetEntries(p.entries)
	} else {
		p.entries = nextEntries
	}
	for _, entry := range p.entries {
		item, _ := entry.(EntityListItem)
		if item.Index == selectedIndex {
			p.searchList.SetSelectedEntry(entry)
			break
		}
	}
	displayState := state
	if p.selectedIndex != selectedIndex {
		p.selectedIndex = selectedIndex
		p.currentState = state
		p.draftDirty = false
	} else if p.draftDirty {
		if transitionPanelStateEqual(state, p.currentState) {
			p.currentState = state
			p.draftDirty = false
		} else {
			displayState = p.currentState
		}
	} else {
		p.currentState = state
	}
	if p.IDInput != nil && !p.IDInput.IsFocused() && p.IDInput.GetText() != displayState.ID {
		p.IDInput.SetText(displayState.ID)
	}
	if p.ToLevelInput != nil && !p.ToLevelInput.IsFocused() && p.ToLevelInput.GetText() != displayState.ToLevel {
		p.ToLevelInput.SetText(displayState.ToLevel)
	}
	if p.LinkedInput != nil && !p.LinkedInput.IsFocused() && p.LinkedInput.GetText() != displayState.LinkedID {
		p.LinkedInput.SetText(displayState.LinkedID)
	}
	p.currentState = displayState
	p.syncDirButtons()
}

func (p *TransitionPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.List)
}

func (p *TransitionPanel) DraftState() (TransitionEditorState, bool) {
	if p == nil || !p.currentState.Selected {
		return TransitionEditorState{}, false
	}
	state := p.currentState
	state.Selected = true
	if p.IDInput != nil {
		state.ID = p.IDInput.GetText()
	}
	if p.ToLevelInput != nil {
		state.ToLevel = p.ToLevelInput.GetText()
	}
	if p.LinkedInput != nil {
		state.LinkedID = p.LinkedInput.GetText()
	}
	if state.EnterDir == "" {
		state.EnterDir = "down"
	}
	return state, true
}

func (p *TransitionPanel) emitEdit() {
	if p == nil || p.syncing || p.callbacks.OnTransitionEdited == nil {
		return
	}
	state, ok := p.DraftState()
	if !ok {
		state = TransitionEditorState{Selected: true, EnterDir: p.currentState.EnterDir}
	}
	if state.EnterDir == "" {
		state.EnterDir = "down"
	}
	p.callbacks.OnTransitionEdited(state)
}

func (p *TransitionPanel) syncDirButtons() {
	for direction, button := range p.DirButtons {
		if button == nil {
			continue
		}
		if direction == p.currentState.EnterDir {
			button.SetImage(p.theme.ActiveButtonImage)
		} else {
			button.SetImage(p.theme.ButtonImage)
		}
	}
}

func transitionPanelStateEqual(left, right TransitionEditorState) bool {
	return left.Selected == right.Selected &&
		left.ID == right.ID &&
		left.ToLevel == right.ToLevel &&
		left.LinkedID == right.LinkedID &&
		left.EnterDir == right.EnterDir
}

type GatePanel struct {
	Root        *widget.Container
	SearchInput *widget.TextInput
	List        *widget.List
	ModeButton  *widget.Button
	GroupInput  *widget.TextInput
	searchList  *SearchableList
	entries     []any
	syncing     bool
	callbacks   LayerCallbacks
}

type TriggerPanel struct {
	Root        *widget.Container
	SearchInput *widget.TextInput
	List        *widget.List
	ModeButton  *widget.Button
	searchList  *SearchableList
	entries     []any
	syncing     bool
}

type BreakableWallPanel struct {
	Root        *widget.Container
	SearchInput *widget.TextInput
	List        *widget.List
	ModeButton  *widget.Button
	searchList  *SearchableList
	entries     []any
	syncing     bool
}

func NewGatePanel(theme *Theme, callbacks LayerCallbacks) *GatePanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &GatePanel{Root: root, callbacks: callbacks}
	root.AddChild(newSectionTitle("Gates", theme))
	panel.ModeButton = newActionButton(theme, "Gates: Off", callbacks.OnGateModeToggled)
	root.AddChild(panel.ModeButton)
	panel.searchList = NewSearchableList(theme, nil, func(entry any) string {
		item, _ := entry.(EntityListItem)
		return item.Label
	}, func(entry any) {
		if panel.syncing || callbacks.OnGateSelected == nil {
			return
		}
		item, ok := entry.(EntityListItem)
		if ok {
			callbacks.OnGateSelected(item.Index)
		}
	})
	panel.List = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.List, 96)
	root.AddChild(panel.searchList.Root)
	root.AddChild(widget.NewText(widget.TextOpts.Text("Group", &theme.Face, theme.MutedTextColor)))
	panel.GroupInput = newEditorTextInput(theme, func(value string) {
		if panel.syncing || callbacks.OnGateEdited == nil {
			return
		}
		callbacks.OnGateEdited(GateEditorState{Selected: true, Group: value})
	})
	root.AddChild(panel.GroupInput)
	return panel
}

func (p *GatePanel) Sync(active bool, items []EntityListItem, selectedIndex int, state GateEditorState) {
	if p == nil {
		return
	}
	label := "Gates: Off"
	if active {
		label = "Gates: On"
	}
	p.ModeButton.SetText(label)
	nextEntries := make([]any, 0, len(items))
	for _, item := range items {
		nextEntries = append(nextEntries, item)
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	if !entriesEqual(p.entries, nextEntries) {
		p.entries = nextEntries
		p.searchList.SetEntries(p.entries)
	} else {
		p.entries = nextEntries
	}
	for _, entry := range p.entries {
		item, _ := entry.(EntityListItem)
		if item.Index == selectedIndex {
			p.searchList.SetSelectedEntry(entry)
			break
		}
	}
	if p.GroupInput != nil && !p.GroupInput.IsFocused() && p.GroupInput.GetText() != state.Group {
		p.GroupInput.SetText(state.Group)
	}
}

func (p *GatePanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.List)
}

func NewTriggerPanel(theme *Theme, callbacks LayerCallbacks) *TriggerPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &TriggerPanel{Root: root}
	root.AddChild(newSectionTitle("Triggers", theme))
	panel.ModeButton = newActionButton(theme, "Triggers: Off", callbacks.OnTriggerModeToggled)
	root.AddChild(panel.ModeButton)
	panel.searchList = NewSearchableList(theme, nil, func(entry any) string {
		item, _ := entry.(EntityListItem)
		return item.Label
	}, func(entry any) {
		if panel.syncing || callbacks.OnTriggerSelected == nil {
			return
		}
		item, ok := entry.(EntityListItem)
		if ok {
			callbacks.OnTriggerSelected(item.Index)
		}
	})
	panel.List = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.List, 96)
	root.AddChild(panel.searchList.Root)
	return panel
}

func (p *TriggerPanel) Sync(active bool, items []EntityListItem, selectedIndex int, _ TriggerEditorState) {
	if p == nil {
		return
	}
	label := "Triggers: Off"
	if active {
		label = "Triggers: On"
	}
	p.ModeButton.SetText(label)
	nextEntries := make([]any, 0, len(items))
	for _, item := range items {
		nextEntries = append(nextEntries, item)
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	if !entriesEqual(p.entries, nextEntries) {
		p.entries = nextEntries
		p.searchList.SetEntries(p.entries)
	} else {
		p.entries = nextEntries
	}
	for _, entry := range p.entries {
		item, _ := entry.(EntityListItem)
		if item.Index == selectedIndex {
			p.searchList.SetSelectedEntry(entry)
			break
		}
	}
	if selectedIndex < 0 {
		p.searchList.SetSelectedEntry(nil)
	}
}

func (p *TriggerPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.List)
}

func NewBreakableWallPanel(theme *Theme, callbacks LayerCallbacks) *BreakableWallPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &BreakableWallPanel{Root: root}
	root.AddChild(newSectionTitle("Breakable Walls", theme))
	panel.ModeButton = newActionButton(theme, "Breakable Walls: Off", callbacks.OnBreakableWallModeToggled)
	root.AddChild(panel.ModeButton)
	panel.searchList = NewSearchableList(theme, nil, func(entry any) string {
		item, _ := entry.(EntityListItem)
		return item.Label
	}, func(entry any) {
		if panel.syncing || callbacks.OnBreakableWallSelected == nil {
			return
		}
		item, ok := entry.(EntityListItem)
		if ok {
			callbacks.OnBreakableWallSelected(item.Index)
		}
	})
	panel.List = panel.searchList.List
	panel.SearchInput = panel.searchList.Input
	setFixedListHeight(panel.List, 96)
	root.AddChild(panel.searchList.Root)
	return panel
}

func (p *BreakableWallPanel) Sync(active bool, items []EntityListItem, selectedIndex int) {
	if p == nil {
		return
	}
	label := "Breakable Walls: Off"
	if active {
		label = "Breakable Walls: On"
	}
	p.ModeButton.SetText(label)
	nextEntries := make([]any, 0, len(items))
	for _, item := range items {
		nextEntries = append(nextEntries, item)
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	if !entriesEqual(p.entries, nextEntries) {
		p.entries = nextEntries
		p.searchList.SetEntries(p.entries)
	} else {
		p.entries = nextEntries
	}
	for _, entry := range p.entries {
		item, _ := entry.(EntityListItem)
		if item.Index == selectedIndex {
			p.searchList.SetSelectedEntry(entry)
			return
		}
	}
	if selectedIndex < 0 {
		p.searchList.SetSelectedEntry(nil)
	}
}

func (p *BreakableWallPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	suppressListAutoScroll(p.List)
}

func (p *InfoPanel) SuppressAutoListScroll() {
	if p == nil {
		return
	}
	if p.LayerPanel != nil {
		p.LayerPanel.SuppressAutoListScroll()
	}
	if p.PrefabPanel != nil {
		p.PrefabPanel.SuppressAutoListScroll()
	}
	if p.EntityPanel != nil {
		p.EntityPanel.SuppressAutoListScroll()
	}
	if p.TransitionPanel != nil {
		p.TransitionPanel.SuppressAutoListScroll()
	}
	if p.GatePanel != nil {
		p.GatePanel.SuppressAutoListScroll()
	}
	if p.TriggerPanel != nil {
		p.TriggerPanel.SuppressAutoListScroll()
	}
	if p.BreakableWallPanel != nil {
		p.BreakableWallPanel.SuppressAutoListScroll()
	}
}

func entriesEqual(left, right []any) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func suppressListAutoScroll(list *widget.List) {
	if list == nil {
		return
	}
	value := reflect.ValueOf(list)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return
	}
	elem := value.Elem()
	focusField := elem.FieldByName("focusIndex")
	prevField := elem.FieldByName("prevFocusIndex")
	if !focusField.IsValid() || !prevField.IsValid() || focusField.Kind() != reflect.Int || prevField.Kind() != reflect.Int {
		return
	}
	focusIndex := focusField.Int()
	if prevField.Int() == focusIndex || !prevField.CanAddr() {
		return
	}
	reflect.NewAt(prevField.Type(), unsafe.Pointer(prevField.UnsafeAddr())).Elem().SetInt(focusIndex)
}

type SearchableList struct {
	Root     *widget.Container
	Input    *widget.TextInput
	List     *widget.List
	label    func(any) string
	entries  []any
	filtered []any
	query    string
}

func NewSearchableList(theme *Theme, entries []any, label func(any) string, onSelected func(any)) *SearchableList {
	searchable := &SearchableList{label: label}
	searchable.Root = widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(6),
		)),
	)
	searchable.Input = newSearchInput(theme, func(value string) {
		searchable.query = value
		searchable.applyFilter()
	})
	searchable.List = newScrollableList(theme, nil, label, onSelected)
	searchable.Root.AddChild(searchable.Input)
	searchable.Root.AddChild(searchable.List)
	searchable.SetEntries(entries)
	return searchable
}

func (s *SearchableList) SetEntries(entries []any) {
	if s == nil {
		return
	}
	s.entries = append(s.entries[:0], entries...)
	s.applyFilter()
}

func (s *SearchableList) SetEnabled(enabled bool) {
	if s == nil {
		return
	}
	if s.Input != nil {
		s.Input.GetWidget().Disabled = !enabled
	}
	if s.List != nil {
		s.List.GetWidget().Disabled = !enabled
	}
}

func (s *SearchableList) SetVisible(visible bool) {
	if s == nil {
		return
	}
	setWidgetVisible(s.Root, visible)
	if s.Input != nil {
		setWidgetVisible(s.Input, visible)
	}
	if s.List != nil {
		setWidgetVisible(s.List, visible)
	}
}

func (s *SearchableList) SetSelectedEntry(entry any) {
	if s == nil || s.List == nil {
		return
	}
	if entry == nil || !containsEntry(s.filtered, entry) {
		s.List.SetSelectedEntry(nil)
		return
	}
	s.List.SetSelectedEntry(entry)
}

func (s *SearchableList) applyFilter() {
	if s == nil || s.List == nil {
		return
	}
	filtered := filterSearchableEntries(s.entries, s.query, s.label)
	s.filtered = filtered
	s.List.SetEntries(filtered)
	s.List.RequestRelayout()
	if selected := s.List.SelectedEntry(); selected != nil && !containsEntry(filtered, selected) {
		s.List.SetSelectedEntry(nil)
	}
}

func filterSearchableEntries(entries []any, query string, label func(any) string) []any {
	query = strings.ToLower(strings.TrimSpace(query))
	filtered := make([]any, 0, len(entries))
	for _, entry := range entries {
		if query == "" || strings.Contains(strings.ToLower(label(entry)), query) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func containsEntry(entries []any, target any) bool {
	for _, entry := range entries {
		if entry == target {
			return true
		}
	}
	return false
}

const scrollableListMaxWidth = 248

func panelTextLayoutData() widget.RowLayoutData {
	return widget.RowLayoutData{
		Position: widget.RowLayoutPositionStart,
		Stretch:  true,
		MaxWidth: scrollableListMaxWidth,
	}
}

func newScrollableList(theme *Theme, entries []any, label func(any) string, onSelected func(any)) *widget.List {
	list := widget.NewList(
		widget.ListOpts.ContainerOpts(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.ElevateLayer(true))),
		widget.ListOpts.Entries(entries),
		widget.ListOpts.EntryLabelFunc(func(entry any) string { return label(entry) }),
		widget.ListOpts.EntryFontFace(&theme.Face),
		widget.ListOpts.EntryColor(theme.ListEntryColor),
		widget.ListOpts.EntryTextPadding(&widget.Insets{Left: 10, Right: 10, Top: 8, Bottom: 8}),
		widget.ListOpts.EntryTextPosition(widget.TextPositionStart, widget.TextPositionCenter),
		widget.ListOpts.ScrollContainerImage(theme.ScrollImage),
		widget.ListOpts.SliderParams(theme.ListSliderParams),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			if onSelected != nil {
				onSelected(args.Entry)
			}
		}),
	)
	return list
}

func newScrollablePanel(theme *Theme, spacing int) (*widget.Container, *widget.Container, *widget.ScrollContainer) {
	root := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(theme.PanelBackground),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	content := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(spacing),
			widget.RowLayoutOpts.Padding(theme.PanelPadding),
		)),
	)
	scroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(transparentScrollContainerImage()),
		widget.ScrollContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				StretchHorizontal: true,
				StretchVertical:   true,
			}),
		),
	)
	bindVerticalWheelScrolling(scroll, content)
	root.AddChild(scroll)
	return root, content, scroll
}

func transparentScrollContainerImage() *widget.ScrollContainerImage {
	mask := euiimage.NewNineSliceColor(color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	transparent := euiimage.NewNineSliceColor(color.NRGBA{})
	return &widget.ScrollContainerImage{
		Idle:     transparent,
		Disabled: transparent,
		Mask:     mask,
	}
}

func bindVerticalWheelScrolling(scroll *widget.ScrollContainer, content widget.PreferredSizeLocateableWidget) {
	if scroll == nil || content == nil {
		return
	}
	scroll.GetWidget().ScrolledEvent.AddHandler(func(args any) {
		eventArgs, ok := args.(*widget.WidgetScrolledEventArgs)
		if !ok {
			return
		}
		_, contentHeight := content.PreferredSize()
		viewHeight := scroll.ViewRect().Dy()
		if contentHeight <= 0 || viewHeight <= 0 || contentHeight <= viewHeight {
			scroll.ScrollTop = 0
			return
		}
		step := float64(viewHeight) / float64(contentHeight)
		scroll.ScrollTop -= eventArgs.Y * (step / 3)
		if scroll.ScrollTop < 0 {
			scroll.ScrollTop = 0
		} else if scroll.ScrollTop > 1 {
			scroll.ScrollTop = 1
		}
	})
}

func setFixedListHeight(list *widget.List, height int) {
	if list == nil {
		return
	}
	_ = applyListHeight(list, height)
}

func applyListHeight(list *widget.List, height int) bool {
	if list == nil {
		return false
	}
	changed := false
	if list.GetWidget().MinHeight != height {
		list.GetWidget().MinHeight = height
		changed = true
	}
	layoutData, ok := list.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok || layoutData.MaxHeight != height || layoutData.MaxWidth != scrollableListMaxWidth || !layoutData.Stretch || layoutData.Position != widget.RowLayoutPositionStart {
		list.GetWidget().LayoutData = widget.RowLayoutData{
			Position:  widget.RowLayoutPositionStart,
			Stretch:   true,
			MaxWidth:  scrollableListMaxWidth,
			MaxHeight: height,
		}
		changed = true
	}
	if changed {
		list.RequestRelayout()
	}
	return changed
}

func newCompactButton(theme *Theme, label string, onClick func()) *widget.Button {
	button := newActionButton(theme, label, onClick)
	button.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	return button
}

func newEditorTextInput(theme *Theme, onChanged func(string)) *widget.TextInput {
	return widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChanged != nil {
				onChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChanged != nil {
				onChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(panelTextLayoutData())),
	)
}

func newSearchInput(theme *Theme, onChanged func(string)) *widget.TextInput {
	return widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Placeholder("Search"),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChanged != nil {
				onChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChanged != nil {
				onChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(panelTextLayoutData())),
	)
}

func newSectionTitle(label string, theme *Theme) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(label, &theme.TitleFace, theme.TextColor),
		widget.TextOpts.MaxWidth(scrollableListMaxWidth),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(panelTextLayoutData())),
	)
}

func newValueText(theme *Theme) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text("", &theme.Face, theme.MutedTextColor),
		widget.TextOpts.MaxWidth(scrollableListMaxWidth),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(panelTextLayoutData())),
	)
}

func newActionButton(theme *Theme, label string, onClick func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonImage),
		widget.ButtonOpts.Text(label, &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if onClick != nil {
				onClick()
			}
		}),
	)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
