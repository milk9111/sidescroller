package editorsystem

import (
	"github.com/hajimehoshi/ebiten/v2"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	editorui "github.com/milk9111/sidescroller/cmd/editor/ui"
	editoruicomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
	"github.com/milk9111/sidescroller/ecs"
)

type EditorUISystem struct {
	ui                            *editorui.EditorUI
	pendingTool                   *editorcomponent.ToolKind
	pendingAsset                  *editorio.AssetInfo
	pendingTileSelection          *model.TileSelection
	pendingPrefab                 *editorio.PrefabInfo
	pendingSaveTarget             *string
	pendingSave                   bool
	pendingLayerSelect            *int
	pendingLayerAdd               bool
	pendingLayerMove              int
	pendingLayerRename            *string
	pendingTogglePhysics          bool
	pendingTogglePhysicsHighlight bool
	pendingToggleAutotile         bool
	pendingEntitySelect           *int
}

func NewEditorUISystem(assets []editorio.AssetInfo, prefabs []editorio.PrefabInfo) (*EditorUISystem, error) {
	system := &EditorUISystem{}
	_ = prefabs
	ui, err := editorui.NewEditorUI(assets, editorui.Callbacks{
		OnToolSelected: func(tool editorcomponent.ToolKind) {
			system.pendingTool = &tool
		},
		OnAssetSelected: func(asset editorio.AssetInfo) {
			copied := asset
			system.pendingAsset = &copied
		},
		OnTileSelected: func(selection model.TileSelection) {
			copied := selection.Normalize()
			system.pendingTileSelection = &copied
		},
		OnPrefabSelected: func(prefab editorio.PrefabInfo) {
			copied := prefab
			system.pendingPrefab = &copied
		},
		OnSaveTargetChanged: func(value string) {
			copied := value
			system.pendingSaveTarget = &copied
		},
		OnSaveRequested: func() {
			system.pendingSave = true
		},
		OnLayerSelected: func(index int) {
			copied := index
			system.pendingLayerSelect = &copied
		},
		OnLayerAdded: func() {
			system.pendingLayerAdd = true
		},
		OnLayerMoved: func(delta int) {
			system.pendingLayerMove = delta
		},
		OnLayerRenamed: func(name string) {
			copied := name
			system.pendingLayerRename = &copied
		},
		OnLayerPhysicsToggled: func() {
			system.pendingTogglePhysics = true
		},
		OnPhysicsHighlightToggled: func() {
			system.pendingTogglePhysicsHighlight = true
		},
		OnAutotileToggled: func() {
			system.pendingToggleAutotile = true
		},
		OnEntitySelected: func(index int) {
			copied := index
			system.pendingEntitySelect = &copied
		},
	})
	if err != nil {
		return nil, err
	}
	system.ui = ui
	return system, nil
}

func (s *EditorUISystem) Update(w *ecs.World) {
	if s == nil || s.ui == nil || w == nil {
		return
	}
	_, session, ok := sessionState(w)
	if !ok || session == nil {
		return
	}
	_, meta, _ := levelMetaState(w)
	width, height := 0, 0
	if meta != nil {
		width = meta.Width
		height = meta.Height
	}
	layers := make([]editoruicomponents.LayerListItem, 0, len(layerEntities(w)))
	for index, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer == nil {
			continue
		}
		layers = append(layers, editoruicomponents.LayerListItem{Index: index, Name: layer.Name, Physics: layer.Physics})
	}
	_, autotile, _ := autotileState(w)
	autotileEnabled := autotile != nil && autotile.Enabled
	_, prefabCatalog, _ := prefabCatalogState(w)
	prefabItems := make([]editoruicomponents.PrefabListItem, 0)
	if prefabCatalog != nil {
		prefabItems = make([]editoruicomponents.PrefabListItem, 0, len(prefabCatalog.Items))
		for _, item := range prefabCatalog.Items {
			prefabItems = append(prefabItems, editoruicomponents.PrefabListItem{Path: item.Path, Name: item.Name, EntityType: item.EntityType})
		}
	}
	_, placement, _ := prefabPlacementState(w)
	_, entitySelection, _ := entitySelectionState(w)
	_, levelEntities, _ := entitiesState(w)
	entityItems := make([]editoruicomponents.EntityListItem, 0)
	selectedEntity := -1
	if entitySelection != nil {
		selectedEntity = entitySelection.SelectedIndex
	}
	if levelEntities != nil {
		entityItems = make([]editoruicomponents.EntityListItem, 0, len(levelEntities.Items))
		for index, item := range levelEntities.Items {
			entityItems = append(entityItems, editoruicomponents.EntityListItem{Index: index, Label: entityLabel(item)})
		}
	}
	selectedIndex := session.SelectedTile.Index
	if autotileEnabled {
		selectedIndex = 0
	}
	selectedPrefabPath := ""
	if placement != nil {
		selectedPrefabPath = placement.SelectedPath
	}
	s.ui.Sync(session.ActiveTool, session.SaveTarget, width, height, session.CurrentLayer, len(layerEntities(w)), layers, autotileEnabled, session.PhysicsHighlight, session.Dirty, prefabItems, selectedPrefabPath, entityItems, selectedEntity, session.SelectedTile.Path, selectedIndex, session.Status)
	s.ui.Update()
	if _, focus, ok := focusState(w); ok && focus != nil {
		focus.SuppressHotkeys = s.ui.AnyInputFocused()
	}
	_, actions, _ := actionState(w)
	if s.pendingSaveTarget != nil {
		session.SaveTarget = *s.pendingSaveTarget
		s.pendingSaveTarget = nil
	}
	if s.pendingTool != nil {
		session.ActiveTool = *s.pendingTool
		s.pendingTool = nil
	}
	if s.pendingAsset != nil {
		session.SelectedTile = model.TileSelection{
			Path:  s.pendingAsset.Name,
			Index: 0,
			TileW: model.DefaultTileSize,
			TileH: model.DefaultTileSize,
		}
		session.Status = "Selected tileset " + s.pendingAsset.Name
		if autotileEnabled {
			session.SelectedTile.Index = 0
		}
		s.pendingAsset = nil
	}
	if s.pendingTileSelection != nil {
		session.SelectedTile = s.pendingTileSelection.Normalize()
		if autotileEnabled {
			session.SelectedTile.Index = 0
		} else {
			session.Status = "Selected tile"
		}
		s.pendingTileSelection = nil
	}
	if s.pendingPrefab != nil && actions != nil {
		actions.SelectPrefab = s.pendingPrefab.Path
		s.pendingPrefab = nil
	}
	if s.pendingSave {
		session.SaveRequested = true
		s.pendingSave = false
	}
	if actions != nil {
		if s.pendingLayerSelect != nil {
			actions.SelectLayer = *s.pendingLayerSelect
			s.pendingLayerSelect = nil
		}
		if s.pendingLayerAdd {
			actions.AddLayer = true
			s.pendingLayerAdd = false
		}
		if s.pendingLayerMove != 0 {
			actions.MoveLayerDelta = s.pendingLayerMove
			s.pendingLayerMove = 0
		}
		if s.pendingLayerRename != nil {
			actions.RenameLayer = *s.pendingLayerRename
			actions.ApplyRename = true
			s.pendingLayerRename = nil
		}
		if s.pendingTogglePhysics {
			actions.ToggleLayerPhysics = true
			s.pendingTogglePhysics = false
		}
		if s.pendingTogglePhysicsHighlight {
			actions.TogglePhysicsHighlight = true
			s.pendingTogglePhysicsHighlight = false
		}
		if s.pendingToggleAutotile {
			actions.ToggleAutotile = true
			s.pendingToggleAutotile = false
		}
		if s.pendingEntitySelect != nil {
			actions.SelectEntity = *s.pendingEntitySelect
			s.pendingEntitySelect = nil
		}
	}
}

func (s *EditorUISystem) Draw(screen *ebiten.Image) {
	if s == nil || s.ui == nil {
		return
	}
	s.ui.Draw(screen)
}

var _ ecs.System = (*EditorUISystem)(nil)
