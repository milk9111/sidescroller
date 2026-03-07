package editorcomponent

import (
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	corecomponent "github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/levels"
)

type ToolKind string

const (
	ToolBrush ToolKind = "brush"
	ToolErase ToolKind = "erase"
	ToolFill  ToolKind = "fill"
	ToolLine  ToolKind = "line"
	ToolSpike ToolKind = "spike"
)

type EditorSession struct {
	ActiveTool       ToolKind
	CurrentLayer     int
	SaveTarget       string
	AssetDir         string
	LoadedLevel      string
	Dirty            bool
	PhysicsHighlight bool
	TransitionMode   bool
	GateMode         bool
	OverviewOpen     bool
	QuitRequested    bool
	SaveRequested    bool
	UndoRequested    bool
	Status           string
	SelectedTile     model.TileSelection
}

type EditorFocus struct {
	SuppressHotkeys bool
}

type EditorClock struct {
	Frame uint64
}

type LevelMeta struct {
	Width       int
	Height      int
	LoadedLevel string
	Dirty       bool
}

type LayerData struct {
	Name         string
	Order        int
	Physics      bool
	Hidden       bool
	Tiles        []int
	TilesetUsage []*levels.TileInfo
}

type LevelEntities struct {
	Items []levels.Entity
}

type TilesetCatalog struct {
	Assets []editorio.AssetInfo
}

type PrefabCatalog struct {
	Items []editorio.PrefabInfo
}

type PrefabPlacementState struct {
	SelectedPath string
	SelectedType string
}

type EntitySelectionState struct {
	SelectedIndex        int
	HoveredIndex         int
	Dragging             bool
	DragOffsetCellX      int
	DragOffsetCellY      int
	DragSnapshotDone     bool
	PropertySnapshotDone bool
}

type EntityClipboardState struct {
	Valid  bool
	Entity levels.Entity
}

type EditorActions struct {
	SelectLayer                        int
	AddLayer                           bool
	MoveLayerDelta                     int
	RenameLayer                        string
	ApplyRename                        bool
	ToggleLayerPhysics                 bool
	ToggleLayerVisibility              bool
	TogglePhysicsHighlight             bool
	ToggleAutotile                     bool
	SelectPrefab                       string
	SelectEntity                       int
	CopySelectedEntity                 bool
	PasteCopiedEntity                  bool
	DeleteSelectedEntity               bool
	ConvertSelectedEntityToPrefabName  string
	ApplyConvertSelectedEntityToPrefab bool
	ClearSelections                    bool
	ToggleTransitionMode               bool
	ToggleGateMode                     bool
	ToggleOverview                     bool
	TransitionID                       string
	TransitionToLevel                  string
	TransitionLinkedID                 string
	TransitionEnterDir                 string
	ApplyTransitionFields              bool
	GateGroup                          string
	ApplyGateFields                    bool
	InspectorFieldComponent            string
	InspectorFieldName                 string
	InspectorFieldValue                string
	ApplyInspectorField                bool
}

type AutotileState struct {
	Enabled     bool
	Remap       []int
	DirtyCells  map[int]map[int]struct{}
	FullRebuild map[int]bool
}

type RawInputState struct {
	MouseX, MouseY int
	WheelX, WheelY float64
	Ctrl, Shift    bool

	LeftDown, RightDown, MiddleDown bool

	LeftJustPressed, LeftJustReleased     bool
	RightJustPressed, RightJustReleased   bool
	MiddleJustPressed, MiddleJustReleased bool
}

type PointerState struct {
	WorldX, WorldY float64
	CellX, CellY   int
	InCanvas       bool
	HasCell        bool
	OverUI         bool
	OverToolbar    bool
	OverLeftPanel  bool
	OverRightPanel bool
}

type CanvasCamera struct {
	X, Y       float64
	Zoom       float64
	ScreenW    float64
	ScreenH    float64
	LeftInset  float64
	RightInset float64
	TopInset   float64
	CanvasX    float64
	CanvasY    float64
	CanvasW    float64
	CanvasH    float64
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

type ToolStroke struct {
	Active     bool
	Tool       ToolKind
	StartCellX int
	StartCellY int
	LastCellX  int
	LastCellY  int
	Touched    map[int]struct{}
	Preview    []GridCell
}

type AreaDragState struct {
	Active              bool
	EntityIndex         int
	Kind                string
	StartCellX          int
	StartCellY          int
	CurrentCellX        int
	CurrentCellY        int
	SnapshotDone        bool
	PropertyEntityIndex int
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

type UndoStack struct {
	Snapshots []model.Snapshot
	Max       int
}

var (
	EditorSessionComponent   = corecomponent.NewComponent[EditorSession]()
	EditorFocusComponent     = corecomponent.NewComponent[EditorFocus]()
	EditorClockComponent     = corecomponent.NewComponent[EditorClock]()
	LevelMetaComponent       = corecomponent.NewComponent[LevelMeta]()
	LayerDataComponent       = corecomponent.NewComponent[LayerData]()
	LevelEntitiesComponent   = corecomponent.NewComponent[LevelEntities]()
	TilesetCatalogComponent  = corecomponent.NewComponent[TilesetCatalog]()
	PrefabCatalogComponent   = corecomponent.NewComponent[PrefabCatalog]()
	PrefabPlacementComponent = corecomponent.NewComponent[PrefabPlacementState]()
	EntitySelectionComponent = corecomponent.NewComponent[EntitySelectionState]()
	EntityClipboardComponent = corecomponent.NewComponent[EntityClipboardState]()
	RawInputStateComponent   = corecomponent.NewComponent[RawInputState]()
	PointerStateComponent    = corecomponent.NewComponent[PointerState]()
	CanvasCameraComponent    = corecomponent.NewComponent[CanvasCamera]()
	ToolStrokeComponent      = corecomponent.NewComponent[ToolStroke]()
	AreaDragStateComponent   = corecomponent.NewComponent[AreaDragState]()
	UndoStackComponent       = corecomponent.NewComponent[UndoStack]()
	EditorActionsComponent   = corecomponent.NewComponent[EditorActions]()
	AutotileStateComponent   = corecomponent.NewComponent[AutotileState]()
	OverviewStateComponent   = corecomponent.NewComponent[OverviewState]()
)
