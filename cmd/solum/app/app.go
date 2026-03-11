package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
	g "github.com/AllenDang/giu"
)

type App struct {
	state *State
}

func New(cfg Config) (*App, error) {
	return &App{state: NewState(cfg)}, nil
}

func (a *App) State() *State {
	return a.state
}

func (a *App) Loop() {
	state := a.state
	state.WindowTitle = state.buildWindowTitle()
	previewPrefabs := strings.Join(state.PrefabPreviewNames(5), ", ")
	if previewPrefabs == "" {
		previewPrefabs = "No prefabs discovered"
	}
	previewEntities := strings.Join(state.EntityPreviewLabels(5), " | ")
	if previewEntities == "" {
		previewEntities = "No entities in level"
	}
	currentLayer := "No layers"
	if len(state.Layers) > 0 {
		layer := state.Layers[state.CurrentLayer]
		currentLayer = fmt.Sprintf("%d: %s", layer.Index+1, layer.Name)
		if layer.Physics {
			currentLayer += " [physics]"
		}
	}
	a.handleHotkeys()
	modeLabel := "Level"
	if state.Overview.Open {
		modeLabel = "Overview"
	}

	layerWidgets := g.Layout{}
	for index, layer := range state.Layers {
		label := fmt.Sprintf("%d. %s (%d tiles)", layer.Index+1, layer.Name, layer.TileCount)
		if index == state.CurrentLayer {
			label = "> " + label
		}
		layerIndex := index
		layerWidgets = append(layerWidgets, g.Button(label).Size(-1, 0).OnClick(func() {
			state.Apply(Action{Kind: ActionSelectLayer, Index: layerIndex})
		}))
	}

	leftPanel := g.Layout{
		g.Label("Solum"),
		g.Label("Milestone 4: overview mode, remaining shortcuts, and UI polish in Solum."),
		g.Label(fmt.Sprintf("Mode: %s", modeLabel)),
		g.Label(fmt.Sprintf("Level: %s", state.DisplayLevelName())),
		g.Label(fmt.Sprintf("Current layer: %s", currentLayer)),
		g.Label(fmt.Sprintf("Size: %s", state.DimensionsLabel())),
		g.Label(state.SummaryLabel()),
		g.Label(fmt.Sprintf("Selected tile: %s", state.SelectedTileLabel())),
		g.Label(fmt.Sprintf("Autotile: %t", state.Autotile.Enabled)),
		g.Label(fmt.Sprintf("Undo entries: %d", len(state.UndoStack))),
		g.InputText(&state.SaveTarget).Label("Save target").Size(-1),
		g.Row(
			g.Button("Save").OnClick(func() {
				state.Apply(Action{Kind: ActionSaveLevel, Target: state.SaveTarget})
			}),
			g.Button("Undo").Disabled(len(state.UndoStack) == 0).OnClick(func() {
				state.Apply(Action{Kind: ActionUndo})
			}),
		),
		g.InputText(&state.LoadTarget).Label("Load level").Size(-1),
		g.Button("Load").OnClick(func() {
			state.Apply(Action{Kind: ActionLoadLevel, Target: state.LoadTarget})
		}),
		g.Label("Layers"),
		g.Child().Border(true).Size(-1, 260).Layout(layerWidgets...),
		g.Row(
			g.Button("Add").OnClick(func() {
				state.Apply(Action{Kind: ActionAddLayer})
			}),
			g.Button("Delete").Disabled(len(state.Layers) <= 1).OnClick(func() {
				state.Apply(Action{Kind: ActionDeleteCurrentLayer})
			}),
		),
		g.Row(
			g.Button("Move Up").Disabled(state.CurrentLayer <= 0).OnClick(func() {
				state.Apply(Action{Kind: ActionMoveCurrentLayer, Delta: -1})
			}),
			g.Button("Move Down").Disabled(state.CurrentLayer >= len(state.Layers)-1).OnClick(func() {
				state.Apply(Action{Kind: ActionMoveCurrentLayer, Delta: 1})
			}),
		),
		g.InputText(&state.LayerNameInput).Label("Layer name").Size(-1),
		g.Row(
			g.Button("Rename").OnClick(func() {
				state.Apply(Action{Kind: ActionRenameCurrentLayer, Name: state.LayerNameInput})
			}),
			g.Button("Toggle Physics").Disabled(len(state.Layers) == 0).OnClick(func() {
				state.Apply(Action{Kind: ActionToggleLayerPhysics})
			}),
		),
		g.Label("Tools"),
		toolButton(state, ToolBrush, "Brush"),
		toolButton(state, ToolErase, "Erase"),
		toolButton(state, ToolFill, "Fill"),
		toolButton(state, ToolBox, "Box"),
		toolButton(state, ToolBoxErase, "Box Erase"),
		toolButton(state, ToolLine, "Line"),
		g.Button(toggleAutotileLabel(state.Autotile.Enabled)).OnClick(func() {
			state.Apply(Action{Kind: ActionToggleAutotile})
		}),
		g.Row(
			g.Button(toggleOverviewLabel(state.Overview.Open)).OnClick(func() {
				state.Apply(Action{Kind: ActionToggleOverview})
			}),
			g.Button("Refresh Overview").OnClick(func() {
				state.Apply(Action{Kind: ActionRefreshOverview})
			}),
		),
	}

	assetWidgets := g.Layout{}
	for _, asset := range state.Assets {
		label := asset.Name
		if state.SelectedTile.Path == asset.Name || state.SelectedTile.Path == asset.Relative {
			label = "> " + label
		}
		path := asset.Name
		assetWidgets = append(assetWidgets, g.Button(label).Size(-1, 0).OnClick(func() {
			state.Apply(Action{Kind: ActionSelectAsset, Path: path})
		}))
	}
	if len(assetWidgets) == 0 {
		assetWidgets = append(assetWidgets, g.Label("No PNG assets discovered"))
	}

	entityWidgets := g.Layout{}
	for _, entityIndex := range currentLayerEntityIndexes(state.Document, state.CurrentLayer) {
		entity := state.Document.Entities[entityIndex]
		label := formatEntityLabel(entityIndex, entity)
		if entityIndex == state.SelectedEntity {
			label = "> " + label
		}
		index := entityIndex
		entityWidgets = append(entityWidgets, g.Button(label).Size(-1, 0).OnClick(func() {
			state.Apply(Action{Kind: ActionSelectEntity, Index: index})
		}))
	}
	if len(entityWidgets) == 0 {
		entityWidgets = append(entityWidgets, g.Label("No entities on the current layer"))
	}

	prefabWidgets := g.Layout{}
	for _, prefab := range state.Prefabs {
		label := fmt.Sprintf("%s (%s)", prefab.Name, prefab.Path)
		if prefab.Path == state.PrefabPlacement.SelectedPath {
			label = "> " + label
		}
		path := prefab.Path
		prefabWidgets = append(prefabWidgets, g.Button(label).Size(-1, 0).OnClick(func() {
			state.Apply(Action{Kind: ActionSelectPrefab, Path: path})
		}))
	}
	if len(prefabWidgets) == 0 {
		prefabWidgets = append(prefabWidgets, g.Label("No prefabs discovered"))
	}

	inspectorWidgets := g.Layout{}
	if !state.Inspector.Active {
		inspectorWidgets = append(inspectorWidgets, g.Label("Select an entity to inspect"))
	} else {
		inspectorWidgets = append(inspectorWidgets,
			g.Label(state.Inspector.EntityLabel),
			g.Label(fmt.Sprintf("Prefab: %s", state.Inspector.PrefabPath)),
		)
		visibleFields := 0
		for _, section := range state.Inspector.Sections {
			if !section.Visible {
				continue
			}
			visibleFields++
			inspectorWidgets = append(inspectorWidgets, g.Label(section.Label))
			for _, field := range section.Fields {
				fieldCopy := field
				key := inspectorFieldKey(field.Component, field.Field)
				if _, ok := state.InspectorInputs[key]; !ok {
					value := field.Value
					state.InspectorInputs[key] = &value
				}
				input := state.InspectorInputs[key]
				inspectorWidgets = append(inspectorWidgets,
					g.Label(fmt.Sprintf("%s [%s]", field.Label, field.TypeLabel)),
					g.Row(
						g.InputText(input).Label(field.Component+"."+field.Field).Size(-160),
						g.Button("Apply").OnClick(func() {
							state.Apply(Action{Kind: ActionEditInspectorField, Name: fieldCopy.Component, Field: fieldCopy.Field, Value: *state.InspectorInputs[key]})
						}),
					),
				)
			}
		}
		if visibleFields == 0 {
			inspectorWidgets = append(inspectorWidgets, g.Label("No editable prefab components found"))
		}
	}

	placeDisabled := strings.TrimSpace(state.PrefabPlacement.SelectedPath) == ""
	placePrefabAction := func() {
		cellX, errX := strconv.Atoi(strings.TrimSpace(state.PlacementCellXInput))
		cellY, errY := strconv.Atoi(strings.TrimSpace(state.PlacementCellYInput))
		if errX != nil || errY != nil {
			state.Status = "Placement coordinates must be integers"
			return
		}
		state.Apply(Action{Kind: ActionPlacePrefab, CellX: cellX, CellY: cellY})
	}

	overviewHovered := strings.TrimSpace(state.Overview.HoveredLevel)
	if overviewHovered == "" {
		overviewHovered = "none"
	}
	overviewIssues := 0
	for _, node := range state.Overview.Nodes {
		overviewIssues += len(node.Diagnostics)
	}
	shortcutLabel := "Ctrl+B/E/F/R/L tool  Ctrl+Shift+R box erase  Ctrl+Z undo  Ctrl+S save  Ctrl+C/V copy paste"
	if state.Overview.Open {
		shortcutLabel = "Z or Esc close overview  Click node to load  Drag node to save layout  Wheel zoom  Middle pan"
	}

	rightPanel := g.Layout{
		g.Label("Tiles"),
		g.Child().Border(true).Size(-1, 180).Layout(assetWidgets...),
		g.Row(
			g.InputText(&state.TileIndexInput).Label("Tile index").Size(120),
			g.Button("Apply Tile").OnClick(func() {
				state.SetTileIndexFromInput(state.TileIndexInput)
			}),
		),
		g.Label("Workspace"),
		g.Label(fmt.Sprintf("Root: %s", state.WorkspaceRoot)),
		g.Label(fmt.Sprintf("Asset dir: %s", state.AssetDir)),
		g.Label(fmt.Sprintf("Autotile remap entries: %d", len(state.AutotileRemap))),
		g.Label("Status"),
		g.Label(strings.TrimSpace(state.Status)),
		g.Label("Overview"),
		g.Label(fmt.Sprintf("Open: %t", state.Overview.Open)),
		g.Label(fmt.Sprintf("Nodes: %d  Edges: %d  Issues: %d", len(state.Overview.Nodes), len(state.Overview.Edges), overviewIssues)),
		g.Label(fmt.Sprintf("Hovered: %s", overviewHovered)),
		g.Label("Preview"),
		g.Label(fmt.Sprintf("Prefab sample: %s", previewPrefabs)),
		g.Label(fmt.Sprintf("Entity sample: %s", previewEntities)),
		g.Label("Prefab Placement"),
		g.Label(fmt.Sprintf("Selected prefab: %s", strings.TrimSpace(state.PrefabPlacement.SelectedPath))),
		g.Row(
			g.InputText(&state.PlacementCellXInput).Label("Cell X").Size(120),
			g.InputText(&state.PlacementCellYInput).Label("Cell Y").Size(120),
			g.Button("Place").Disabled(placeDisabled).OnClick(placePrefabAction),
		),
		g.Label("Prefab Catalog"),
		g.Child().Border(true).Size(-1, 220).Layout(prefabWidgets...),
		g.Label("Entities On Current Layer"),
		g.Child().Border(true).Size(-1, 220).Layout(entityWidgets...),
		g.Row(
			g.Button("Clear Selection").OnClick(func() {
				state.Apply(Action{Kind: ActionClearEntity})
			}),
			g.Button("Delete").Disabled(state.SelectedEntity < 0).OnClick(func() {
				state.Apply(Action{Kind: ActionDeleteEntity})
			}),
		),
		g.Row(
			g.Button("Copy").Disabled(state.SelectedEntity < 0).OnClick(func() {
				state.Apply(Action{Kind: ActionCopyEntity})
			}),
			g.Button("Paste").Disabled(!state.Clipboard.Valid).OnClick(func() {
				state.Apply(Action{Kind: ActionPasteEntity})
			}),
		),
		g.InputText(&state.ConvertPrefabTarget).Label("Convert to prefab").Size(-1),
		g.Button("Convert Selected Entity").Disabled(state.SelectedEntity < 0).OnClick(func() {
			state.Apply(Action{Kind: ActionConvertToPrefab, Target: state.ConvertPrefabTarget})
		}),
		g.Label("Inspector"),
		g.Child().Border(true).Size(-1, 360).Layout(inspectorWidgets...),
		g.Label("Shortcuts"),
		g.Label(shortcutLabel),
		g.Label("Notes"),
		g.Label("Viewport supports paint, pan, zoom, sampling, autotile recompute, and level overview mode."),
	}

	viewportPanel := g.Layout{
		g.Label(fmt.Sprintf("Viewport (%s)", modeLabel)),
		a.buildViewportWidget(),
	}

	topBar := g.Layout{
		g.Row(
			g.Button("Save").OnClick(func() {
				state.Apply(Action{Kind: ActionSaveLevel, Target: state.SaveTarget})
			}),
			g.Button("Undo").Disabled(len(state.UndoStack) == 0).OnClick(func() {
				state.Apply(Action{Kind: ActionUndo})
			}),
			g.Button(toggleOverviewLabel(state.Overview.Open)).OnClick(func() {
				state.Apply(Action{Kind: ActionToggleOverview})
			}),
			g.Button("Refresh Overview").OnClick(func() {
				state.Apply(Action{Kind: ActionRefreshOverview})
			}),
			g.Label(fmt.Sprintf("Active: %s", state.ActiveTool)),
			g.Label(fmt.Sprintf("Layer %d/%d", state.CurrentLayer+1, maxInt(1, len(state.Layers)))),
			g.Label(fmt.Sprintf("Dirty: %t", state.Dirty)),
		),
	}

	g.SingleWindow().Layout(
		topBar,
		g.SplitLayout(
			g.DirectionVertical,
			&state.SidebarWidth,
			leftPanel,
			g.SplitLayout(g.DirectionVertical, &state.RightSidebarWidth, g.Child().Border(true).Layout(viewportPanel...), g.Child().Border(false).Layout(rightPanel...)),
		),
	)
}

func (a *App) handleHotkeys() {
	state := a.state
	io := imgui.CurrentIO()
	if io.WantTextInput() {
		return
	}
	if io.KeyCtrl() {
		switch {
		case g.IsKeyPressed(g.KeyB):
			state.Apply(Action{Kind: ActionSelectTool, Name: string(ToolBrush)})
		case g.IsKeyPressed(g.KeyE):
			state.Apply(Action{Kind: ActionSelectTool, Name: string(ToolErase)})
		case g.IsKeyPressed(g.KeyF):
			state.Apply(Action{Kind: ActionSelectTool, Name: string(ToolFill)})
		case g.IsKeyPressed(g.KeyR):
			tool := ToolBox
			if io.KeyShift() {
				tool = ToolBoxErase
			}
			state.Apply(Action{Kind: ActionSelectTool, Name: string(tool)})
		case g.IsKeyPressed(g.KeyL):
			state.Apply(Action{Kind: ActionSelectTool, Name: string(ToolLine)})
		case g.IsKeyPressed(g.KeyZ):
			state.Apply(Action{Kind: ActionUndo})
		case g.IsKeyPressed(g.KeyS):
			state.Apply(Action{Kind: ActionSaveLevel, Target: state.SaveTarget})
		case g.IsKeyPressed(g.KeyC):
			state.Apply(Action{Kind: ActionCopyEntity})
		case g.IsKeyPressed(g.KeyV):
			state.Apply(Action{Kind: ActionPasteEntity})
		}
		return
	}
	if g.IsKeyPressed(g.KeyZ) {
		state.Apply(Action{Kind: ActionToggleOverview})
		return
	}
	if state.Overview.Open {
		if g.IsKeyPressed(g.KeyEscape) {
			state.Apply(Action{Kind: ActionToggleOverview})
		}
		return
	}
	if g.IsKeyPressed(g.KeyQ) {
		state.Apply(Action{Kind: ActionSelectLayer, Index: state.CurrentLayer - 1})
	}
	if g.IsKeyPressed(g.KeyE) {
		state.Apply(Action{Kind: ActionSelectLayer, Index: state.CurrentLayer + 1})
	}
	if g.IsKeyPressed(g.KeyN) {
		state.Apply(Action{Kind: ActionAddLayer})
	}
	if g.IsKeyPressed(g.KeyH) {
		state.Apply(Action{Kind: ActionToggleLayerPhysics})
	}
	if g.IsKeyPressed(g.KeyT) {
		state.Apply(Action{Kind: ActionToggleAutotile})
	}
	if g.IsKeyPressed(g.KeyDelete) || g.IsKeyPressed(g.KeyBackspace) {
		if io.KeyShift() && len(state.Layers) > 1 {
			state.Apply(Action{Kind: ActionDeleteCurrentLayer})
		} else {
			state.Apply(Action{Kind: ActionDeleteEntity})
		}
	}
	if g.IsKeyPressed(g.KeyEscape) {
		state.Apply(Action{Kind: ActionClearEntity})
	}
}

func toolButton(state *State, tool ToolKind, label string) g.Widget {
	if state.ActiveTool == tool {
		label = "> " + label
	}
	return g.Button(label).Size(-1, 0).OnClick(func() {
		state.Apply(Action{Kind: ActionSelectTool, Name: string(tool)})
	})
}

func toggleAutotileLabel(enabled bool) string {
	if enabled {
		return "Disable Autotile"
	}
	return "Enable Autotile"
}

func toggleOverviewLabel(open bool) string {
	if open {
		return "Close Overview"
	}
	return "Open Overview"
}

func mouseWheelY() float64 {
	return float64(imgui.CurrentIO().MouseWheel())
}
