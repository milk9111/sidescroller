package app

import (
	"fmt"
	"strings"

	coreio "github.com/milk9111/sidescroller/internal/editorcore/io"
	corelevelops "github.com/milk9111/sidescroller/internal/editorcore/levelops"
	coremodel "github.com/milk9111/sidescroller/internal/editorcore/model"
	"github.com/milk9111/sidescroller/levels"
)

type Config struct {
	WorkspaceRoot string
	AssetDir      string
	LevelName     string
	SaveTarget    string
	Level         *coremodel.LevelDocument
	Assets        []coreio.AssetInfo
	Prefabs       []coreio.PrefabInfo
	AutotileRemap []int
}

type LayerState struct {
	Index     int
	Name      string
	Physics   bool
	TileCount int
}

type ToolKind string

const (
	ToolBrush    ToolKind = "brush"
	ToolErase    ToolKind = "erase"
	ToolFill     ToolKind = "fill"
	ToolBox      ToolKind = "box"
	ToolBoxErase ToolKind = "box_erase"
	ToolLine     ToolKind = "line"
)

type ActionKind string

const (
	ActionSelectLayer        ActionKind = "select-layer"
	ActionAddLayer           ActionKind = "add-layer"
	ActionDeleteCurrentLayer ActionKind = "delete-current-layer"
	ActionMoveCurrentLayer   ActionKind = "move-current-layer"
	ActionRenameCurrentLayer ActionKind = "rename-current-layer"
	ActionToggleLayerPhysics ActionKind = "toggle-layer-physics"
	ActionSaveLevel          ActionKind = "save-level"
	ActionLoadLevel          ActionKind = "load-level"
	ActionToggleOverview     ActionKind = "toggle-overview"
	ActionRefreshOverview    ActionKind = "refresh-overview"
	ActionUndo               ActionKind = "undo"
	ActionSelectEntity       ActionKind = "select-entity"
	ActionClearEntity        ActionKind = "clear-entity"
	ActionSelectPrefab       ActionKind = "select-prefab"
	ActionPlacePrefab        ActionKind = "place-prefab"
	ActionDeleteEntity       ActionKind = "delete-entity"
	ActionCopyEntity         ActionKind = "copy-entity"
	ActionPasteEntity        ActionKind = "paste-entity"
	ActionEditInspectorField ActionKind = "edit-inspector-field"
	ActionConvertToPrefab    ActionKind = "convert-to-prefab"
	ActionSelectTool         ActionKind = "select-tool"
	ActionToggleAutotile     ActionKind = "toggle-autotile"
	ActionSelectAsset        ActionKind = "select-asset"
	ActionSetTileIndex       ActionKind = "set-tile-index"
)

type Action struct {
	Kind   ActionKind
	Index  int
	Delta  int
	Name   string
	Target string
	Path   string
	Value  string
	Field  string
	CellX  int
	CellY  int
}

type PrefabPlacementState struct {
	SelectedPath string
	SelectedType string
}

type PointerState struct {
	WorldX     float64
	WorldY     float64
	CellX      int
	CellY      int
	InCanvas   bool
	HasCell    bool
	OverUI     bool
	OverCanvas bool
}

type CameraState struct {
	X          float64
	Y          float64
	Zoom       float64
	CanvasX    float64
	CanvasY    float64
	CanvasW    float64
	CanvasH    float64
	LeftInset  float64
	RightInset float64
	TopInset   float64
	PanActive  bool
	PanMouseX  int
	PanMouseY  int
	PanStartX  float64
	PanStartY  float64
}

type GridCell struct {
	X int
	Y int
}

type ToolStrokeState struct {
	Active      bool
	Tool        ToolKind
	StartCellX  int
	StartCellY  int
	LastCellX   int
	LastCellY   int
	SnapshotLen int
	Changed     bool
	Touched     map[int]struct{}
	Preview     []GridCell
}

type MoveSelectionState struct {
	Active   bool
	Moving   bool
	Width    int
	Height   int
	DestMinX int
	DestMinY int
}

type AreaDragState struct {
	Active      bool
	EntityIndex int
	Kind        string
}

type OverviewNode struct {
	Level       string
	DisplayName string
	X           float64
	Y           float64
	W           float64
	H           float64
	Diagnostics []string
	HasManual   bool
}

type OverviewEdge struct {
	From      string
	To        string
	Direction string
	Warning   bool
}

type OverviewState struct {
	Open          bool
	Nodes         []OverviewNode
	Edges         []OverviewEdge
	HoveredLevel  string
	DraggingLevel string
	PressedLevel  string
	DragOffsetX   float64
	DragOffsetY   float64
	PanX          float64
	PanY          float64
	Zoom          float64
	PanActive     bool
	PanMouseX     int
	PanMouseY     int
	PanStartX     float64
	PanStartY     float64
	DragMoved     bool
	NeedsRefresh  bool
	NeedsPersist  bool
	LoadLevel     string
}

type AutotileState struct {
	Enabled     bool
	DirtyCells  map[int]map[int]struct{}
	FullRebuild map[int]bool
}

type State struct {
	WorkspaceRoot       string
	AssetDir            string
	LoadedLevel         string
	SaveTarget          string
	LoadTarget          string
	Document            coremodel.LevelDocument
	SelectedTile        coremodel.TileSelection
	Assets              []coreio.AssetInfo
	Prefabs             []coreio.PrefabInfo
	AutotileRemap       []int
	Autotile            AutotileState
	Layers              []LayerState
	Status              string
	Dirty               bool
	CurrentLayer        int
	ActiveTool          ToolKind
	SelectedEntity      int
	UndoStack           []coremodel.Snapshot
	UndoLimit           int
	LayerNameInput      string
	SidebarWidth        float32
	RightSidebarWidth   float32
	PrefabPlacement     PrefabPlacementState
	Pointer             PointerState
	Camera              CameraState
	ToolStroke          ToolStrokeState
	MoveSelection       MoveSelectionState
	AreaDrag            AreaDragState
	Overview            OverviewState
	WindowTitle         string
	Clipboard           EntityClipboardState
	Inspector           InspectorState
	InspectorInputs     map[string]*string
	PlacementCellXInput string
	PlacementCellYInput string
	ConvertPrefabTarget string
	TileIndexInput      string
}

func NewState(cfg Config) *State {
	doc := cfg.Level
	if doc == nil {
		doc = coremodel.NewLevelDocument(40, 22)
	}
	assetNames := make([]string, 0, len(cfg.Assets))
	for _, asset := range cfg.Assets {
		assetNames = append(assetNames, asset.Name)
	}
	selection := coremodel.InferSelection(doc, assetNames)
	state := &State{
		WorkspaceRoot:       cfg.WorkspaceRoot,
		AssetDir:            cfg.AssetDir,
		LoadedLevel:         strings.TrimSpace(cfg.LevelName),
		SaveTarget:          cfg.SaveTarget,
		LoadTarget:          cfg.SaveTarget,
		Document:            doc.Clone(),
		SelectedTile:        selection,
		Assets:              append([]coreio.AssetInfo(nil), cfg.Assets...),
		Prefabs:             append([]coreio.PrefabInfo(nil), cfg.Prefabs...),
		AutotileRemap:       append([]int(nil), cfg.AutotileRemap...),
		Autotile:            AutotileState{DirtyCells: make(map[int]map[int]struct{}), FullRebuild: make(map[int]bool)},
		Status:              "Solum milestone 4 ready",
		CurrentLayer:        0,
		ActiveTool:          ToolBrush,
		SelectedEntity:      -1,
		UndoLimit:           100,
		SidebarWidth:        360,
		RightSidebarWidth:   420,
		Camera:              CameraState{Zoom: 1},
		Overview:            OverviewState{Zoom: 1, NeedsRefresh: true},
		InspectorInputs:     make(map[string]*string),
		PlacementCellXInput: "0",
		PlacementCellYInput: "0",
		TileIndexInput:      fmt.Sprintf("%d", selection.Index),
	}
	state.syncDerivedState(true)
	state.WindowTitle = state.buildWindowTitle()
	return state
}

func (s *State) buildWindowTitle() string {
	label := s.DisplayLevelName()
	if s.Dirty {
		label += " *"
	}
	return fmt.Sprintf("Solum - %s", label)
}

func (s *State) DisplayLevelName() string {
	if strings.TrimSpace(s.LoadedLevel) != "" {
		return s.LoadedLevel
	}
	if strings.TrimSpace(s.SaveTarget) != "" {
		return s.SaveTarget
	}
	return "untitled.json"
}

func (s *State) DimensionsLabel() string {
	return fmt.Sprintf("%d x %d", s.Document.Width, s.Document.Height)
}

func (s *State) SummaryLabel() string {
	return fmt.Sprintf(
		"Layers: %d  Entities: %d  Assets: %d  Prefabs: %d  Tool: %s",
		len(s.Document.Layers),
		len(s.Document.Entities),
		len(s.Assets),
		len(s.Prefabs),
		s.ActiveTool,
	)
}

func (s *State) SelectedTileLabel() string {
	selection := s.SelectedTile.Normalize()
	if strings.TrimSpace(selection.Path) == "" {
		return "No tile selected"
	}
	return fmt.Sprintf("%s @ %d", selection.Path, selection.Index)
}

func (s *State) PrefabPreviewNames(limit int) []string {
	if limit <= 0 || limit > len(s.Prefabs) {
		limit = len(s.Prefabs)
	}
	names := make([]string, 0, limit)
	for _, prefab := range s.Prefabs[:limit] {
		names = append(names, prefab.Name)
	}
	return names
}

func (s *State) EntityPreviewLabels(limit int) []string {
	if limit <= 0 || limit > len(s.Document.Entities) {
		limit = len(s.Document.Entities)
	}
	labels := make([]string, 0, limit)
	for index, entity := range s.Document.Entities[:limit] {
		labels = append(labels, formatEntityLabel(index, entity))
	}
	return labels
}

func (s *State) Apply(action Action) {
	if s == nil {
		return
	}
	switch action.Kind {
	case ActionSelectLayer:
		s.CurrentLayer = corelevelops.ClampLayerIndex(s.Document, action.Index)
		s.syncDerivedState(true)
		s.Status = fmt.Sprintf("Selected layer %d", s.CurrentLayer+1)
	case ActionAddLayer:
		s.pushSnapshot("layer-add")
		s.CurrentLayer = corelevelops.AddLayer(&s.Document)
		s.Dirty = true
		s.syncDerivedState(true)
		s.Status = "Added layer"
	case ActionDeleteCurrentLayer:
		if len(s.Document.Layers) <= 1 {
			s.Status = "Cannot delete last layer"
			return
		}
		s.pushSnapshot("layer-delete")
		if corelevelops.DeleteLayer(&s.Document, s.CurrentLayer) {
			s.Dirty = true
			s.SelectedEntity = -1
			s.syncDerivedState(true)
			s.Status = "Deleted layer"
		}
	case ActionMoveCurrentLayer:
		next := corelevelops.ClampLayerIndex(s.Document, s.CurrentLayer+action.Delta)
		if next == s.CurrentLayer {
			return
		}
		s.pushSnapshot("layer-move")
		if corelevelops.MoveLayer(&s.Document, s.CurrentLayer, next) {
			s.CurrentLayer = next
			s.Dirty = true
			s.syncDerivedState(true)
			s.Status = "Moved layer"
		}
	case ActionRenameCurrentLayer:
		s.pushSnapshot("layer-rename")
		if corelevelops.RenameLayer(&s.Document, s.CurrentLayer, action.Name) {
			s.Dirty = true
			s.syncDerivedState(true)
			s.Status = "Renamed layer"
		} else if strings.TrimSpace(action.Name) == "" {
			s.Status = "Layer name cannot be empty"
		}
	case ActionToggleLayerPhysics:
		s.pushSnapshot("layer-physics")
		enabled, ok := corelevelops.ToggleLayerPhysics(&s.Document, s.CurrentLayer)
		if ok {
			s.Dirty = true
			s.syncDerivedState(false)
			if enabled {
				s.Status = "Layer physics enabled"
			} else {
				s.Status = "Layer physics disabled"
			}
		}
	case ActionSaveLevel:
		s.saveLevel(action.Target)
	case ActionLoadLevel:
		s.loadLevel(action.Target)
	case ActionToggleOverview:
		s.toggleOverview()
	case ActionRefreshOverview:
		s.refreshOverview()
	case ActionUndo:
		s.undo()
	case ActionSelectEntity:
		s.selectEntity(action.Index)
	case ActionClearEntity:
		s.clearEntitySelection()
	case ActionSelectPrefab:
		s.selectPrefab(action.Path)
	case ActionPlacePrefab:
		s.placeSelectedPrefab(action.CellX, action.CellY)
	case ActionDeleteEntity:
		s.deleteSelectedEntity()
	case ActionCopyEntity:
		s.copySelectedEntity()
	case ActionPasteEntity:
		s.pasteCopiedEntity()
	case ActionEditInspectorField:
		s.editInspectorField(action.Name, action.Field, action.Value)
	case ActionConvertToPrefab:
		s.convertSelectedEntityToPrefab(action.Target)
	case ActionSelectTool:
		s.selectTool(ToolKind(action.Name))
	case ActionToggleAutotile:
		s.toggleAutotile()
	case ActionSelectAsset:
		s.selectAsset(action.Path)
	case ActionSetTileIndex:
		s.setSelectedTileIndex(action.Index)
	}
	s.WindowTitle = s.buildWindowTitle()
}

func (s *State) saveLevel(target string) {
	if s == nil {
		return
	}
	target = strings.TrimSpace(target)
	if target == "" {
		target = s.SaveTarget
	}
	if target == "" {
		target = "untitled.json"
	}
	working := s.Document.Clone()
	if corelevelops.EnsureUniqueEntityIDs(&working) {
		s.Document = working.Clone()
		s.syncDerivedState(false)
		s.Dirty = true
	}
	normalized, err := coreio.SaveLevel(s.WorkspaceRoot, target, &s.Document)
	if err != nil {
		s.Status = fmt.Sprintf("Save failed: %v", err)
		return
	}
	s.SaveTarget = normalized
	s.LoadedLevel = normalized
	s.LoadTarget = normalized
	s.Overview.NeedsRefresh = true
	s.Dirty = false
	s.Status = "Saved levels/" + normalized
}

func (s *State) loadLevel(target string) {
	if s == nil {
		return
	}
	target = strings.TrimSpace(target)
	if target == "" {
		target = s.LoadTarget
	}
	if target == "" {
		s.Status = "Enter a level name to load"
		return
	}
	doc, normalized, err := coreio.LoadLevel(s.WorkspaceRoot, target)
	if err != nil {
		s.Status = fmt.Sprintf("Load failed: %v", err)
		return
	}
	assetNames := make([]string, 0, len(s.Assets))
	for _, asset := range s.Assets {
		assetNames = append(assetNames, asset.Name)
	}
	s.Document = doc.Clone()
	s.LoadedLevel = normalized
	s.SaveTarget = normalized
	s.LoadTarget = normalized
	s.SelectedTile = coremodel.InferSelection(doc, assetNames)
	s.Dirty = false
	s.SelectedEntity = -1
	s.UndoStack = nil
	s.Overview.Open = false
	s.Overview.NeedsRefresh = true
	s.syncDerivedState(true)
	s.Status = "Loaded levels/" + normalized
}

func (s *State) undo() {
	if s == nil {
		return
	}
	if len(s.UndoStack) == 0 {
		s.Status = "Undo stack empty"
		return
	}
	last := s.UndoStack[len(s.UndoStack)-1]
	s.UndoStack = s.UndoStack[:len(s.UndoStack)-1]
	s.Document = last.Level.Clone()
	s.CurrentLayer = last.CurrentLayer
	s.SaveTarget = last.SaveTarget
	s.LoadedLevel = last.LoadedLevel
	s.LoadTarget = last.SaveTarget
	s.SelectedTile = last.SelectedTile.Normalize()
	s.Dirty = true
	s.SelectedEntity = -1
	s.ToolStroke = ToolStrokeState{}
	s.syncDerivedState(true)
	s.Status = "Undo applied"
}

func (s *State) pushSnapshot(reason string) {
	if s == nil {
		return
	}
	snapshot := coremodel.Snapshot{
		Level:         s.Document.Clone(),
		CurrentLayer:  s.CurrentLayer,
		SaveTarget:    s.SaveTarget,
		LoadedLevel:   s.LoadedLevel,
		SelectedTile:  s.SelectedTile.Normalize(),
		StatusMessage: reason,
	}
	s.UndoStack = append(s.UndoStack, snapshot)
	if s.UndoLimit <= 0 {
		s.UndoLimit = 100
	}
	if len(s.UndoStack) > s.UndoLimit {
		s.UndoStack = append([]coremodel.Snapshot(nil), s.UndoStack[len(s.UndoStack)-s.UndoLimit:]...)
	}
}

func (s *State) syncDerivedState(resetLayerInput bool) {
	if s == nil {
		return
	}
	s.Layers = buildLayerState(s.Document)
	s.CurrentLayer = corelevelops.ClampLayerIndex(s.Document, s.CurrentLayer)
	if resetLayerInput {
		if len(s.Document.Layers) > 0 {
			s.LayerNameInput = s.Document.Layers[s.CurrentLayer].Name
		} else {
			s.LayerNameInput = ""
		}
	}
	s.syncEntityDerivedState()
	s.SelectedTile = s.SelectedTile.Normalize()
	s.TileIndexInput = fmt.Sprintf("%d", s.SelectedTile.Index)
	s.WindowTitle = s.buildWindowTitle()
}

func buildLayerState(doc coremodel.LevelDocument) []LayerState {
	layers := make([]LayerState, 0, len(doc.Layers))
	for index, layer := range doc.Layers {
		tileCount := 0
		for _, tile := range layer.Tiles {
			if tile != 0 {
				tileCount++
			}
		}
		layers = append(layers, LayerState{
			Index:     index,
			Name:      layer.Name,
			Physics:   layer.Physics,
			TileCount: tileCount,
		})
	}
	return layers
}

func formatEntityLabel(index int, entity levels.Entity) string {
	label := strings.TrimSpace(entity.ID)
	if label == "" {
		label = strings.TrimSpace(entity.Type)
	}
	if label == "" {
		label = "entity"
	}
	return fmt.Sprintf("%d: %s (%d, %d)", index, label, entity.X, entity.Y)
}
