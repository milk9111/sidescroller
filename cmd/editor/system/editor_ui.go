package editorsystem

import (
	"strings"

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
	syncedEntitySelection         int
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
	pendingToggleLayerVisibility  bool
	pendingTogglePhysicsHighlight bool
	pendingToggleAutotile         bool
	pendingEntitySelect           *int
	pendingConvertPrefabName      *string
	pendingTransitionModeToggle   bool
	pendingGateModeToggle         bool
	pendingTransitionSelect       *int
	pendingGateSelect             *int
	pendingTransitionEdit         *editoruicomponents.TransitionEditorState
	pendingGateEdit               *editoruicomponents.GateEditorState
	pendingInspectorEdit          *editoruicomponents.InspectorFieldEdit
}

func NewEditorUISystem(assets []editorio.AssetInfo, prefabs []editorio.PrefabInfo) (*EditorUISystem, error) {
	system := &EditorUISystem{syncedEntitySelection: -1}
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
		OnLayerVisibilityToggled: func() {
			system.pendingToggleLayerVisibility = true
		},
		OnPhysicsHighlightToggled: func() {
			system.pendingTogglePhysicsHighlight = true
		},
		OnAutotileToggled: func() {
			system.pendingToggleAutotile = true
		},
		OnEntitySelected: func(index int) {
			if index == system.syncedEntitySelection {
				return
			}
			copied := index
			system.pendingEntitySelect = &copied
		},
		OnConvertToPrefabConfirmed: func(name string) {
			copied := name
			system.pendingConvertPrefabName = &copied
		},
		OnTransitionModeToggled: func() {
			system.pendingTransitionModeToggle = true
		},
		OnGateModeToggled: func() {
			system.pendingGateModeToggle = true
		},
		OnTransitionSelected: func(index int) {
			copied := index
			system.pendingTransitionSelect = &copied
		},
		OnGateSelected: func(index int) {
			copied := index
			system.pendingGateSelect = &copied
		},
		OnTransitionEdited: func(state editoruicomponents.TransitionEditorState) {
			copied := state
			system.pendingTransitionEdit = &copied
		},
		OnGateEdited: func(state editoruicomponents.GateEditorState) {
			copied := state
			system.pendingGateEdit = &copied
		},
		OnInspectorFieldEdited: func(edit editoruicomponents.InspectorFieldEdit) {
			copied := edit
			system.pendingInspectorEdit = &copied
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
		layers = append(layers, editoruicomponents.LayerListItem{Index: index, Name: layer.Name, Physics: layer.Physics, Visible: layerVisible(layer)})
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
	transitionItems := make([]editoruicomponents.EntityListItem, 0)
	gateItems := make([]editoruicomponents.EntityListItem, 0)
	transitionState := editoruicomponents.TransitionEditorState{EnterDir: "down"}
	gateState := editoruicomponents.GateEditorState{Group: "boss_gate"}
	inspectorState := editoruicomponents.InspectorState{}
	selectedEntity := -1
	if entitySelection != nil {
		selectedEntity = entitySelection.SelectedIndex
	}
	s.syncedEntitySelection = selectedEntity
	selectedIndex := session.SelectedTile.Index
	if autotileEnabled {
		selectedIndex = 0
	}
	selectedPrefabPath := ""
	if placement != nil {
		selectedPrefabPath = placement.SelectedPath
	}
	if levelEntities != nil {
		entityItems = make([]editoruicomponents.EntityListItem, 0, len(levelEntities.Items))
		for index, item := range levelEntities.Items {
			if !entitySelectableOnCurrentLayer(w, session, item) {
				continue
			}
			entityItems = append(entityItems, editoruicomponents.EntityListItem{Index: index, Label: entityLabel(item)})
			if isTransitionEntity(item) {
				transitionItems = append(transitionItems, editoruicomponents.EntityListItem{Index: index, Label: entityLabel(item)})
			}
			if isGateEntity(item) {
				gateItems = append(gateItems, editoruicomponents.EntityListItem{Index: index, Label: entityLabel(item)})
			}
		}
		// If a transition was selected from the transition list but the ECS selection
		// hasn't been applied yet, prefill the transition editor from that item so
		// the form updates immediately for the user.
		prefillTransition := editoruicomponents.TransitionEditorState{EnterDir: "down"}
		if s.pendingTransitionSelect != nil {
			idx := *s.pendingTransitionSelect
			if idx >= 0 && idx < len(levelEntities.Items) {
				sel := levelEntities.Items[idx]
				if isTransitionEntity(sel) {
					prefillTransition = editoruicomponents.TransitionEditorState{
						Selected: true,
						ID:       entityStringProp(sel, "id"),
						ToLevel:  entityStringProp(sel, "to_level"),
						LinkedID: entityStringProp(sel, "linked_id"),
						EnterDir: entityStringProp(sel, "enter_dir"),
					}
					if prefillTransition.ID == "" {
						prefillTransition.ID = strings.TrimSpace(sel.ID)
					}
					if prefillTransition.EnterDir == "" {
						prefillTransition.EnterDir = "down"
					}
				}
			}
		}
		transitionState = prefillTransition
		gateState = editoruicomponents.GateEditorState{Group: "boss_gate"}
		// Only build the transition/gate editor state from the current ECS selection
		// if there isn't a pending transition list selection to prefill the form.
		if s.pendingTransitionSelect == nil && entitySelection != nil && entitySelection.SelectedIndex >= 0 && entitySelection.SelectedIndex < len(levelEntities.Items) {
			selected := levelEntities.Items[entitySelection.SelectedIndex]
			if isTransitionEntity(selected) {
				transitionState = editoruicomponents.TransitionEditorState{
					Selected: true,
					ID:       entityStringProp(selected, "id"),
					ToLevel:  entityStringProp(selected, "to_level"),
					LinkedID: entityStringProp(selected, "linked_id"),
					EnterDir: entityStringProp(selected, "enter_dir"),
				}
				if transitionState.ID == "" {
					transitionState.ID = strings.TrimSpace(selected.ID)
				}
				if transitionState.EnterDir == "" {
					transitionState.EnterDir = "down"
				}
			}
			if isGateEntity(selected) {
				gateState = editoruicomponents.GateEditorState{Selected: true, Group: entityStringProp(selected, "group")}
				if gateState.Group == "" {
					gateState.Group = "boss_gate"
				}
			}
		}
		inspectorState = buildInspectorState(prefabCatalog, levelEntities, selectedEntity)
	}
	s.ui.Sync(session.ActiveTool, session.SaveTarget, width, height, session.CurrentLayer, len(layerEntities(w)), layers, autotileEnabled, session.PhysicsHighlight, session.Dirty, prefabItems, selectedPrefabPath, entityItems, selectedEntity, session.TransitionMode, session.GateMode, transitionItems, gateItems, transitionState, gateState, session.SelectedTile.Path, selectedIndex, session.Status, inspectorState)
	s.ui.Update()
	if _, camera, ok := cameraState(w); ok && camera != nil {
		metrics := s.ui.LayoutMetrics(int(camera.ScreenW), int(camera.ScreenH))
		camera.LeftInset = metrics.LeftInset
		camera.RightInset = metrics.RightInset
		camera.TopInset = metrics.TopInset
		layoutPanels(camera)
		if _, input, ok := rawInputState(w); ok && input != nil {
			if _, pointer, ok := pointerState(w); ok && pointer != nil {
				refreshPointerFromCamera(pointer, input, camera, meta)
			}
		}
	}
	if _, pointer, ok := pointerState(w); ok && pointer != nil {
		mouseX, mouseY := ebiten.CursorPosition()
		pointer.OverUI = s.ui.PointerOverUI(mouseX, mouseY)
		if pointer.OverUI {
			pointer.InCanvas = false
			pointer.HasCell = false
		}
	}
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
		if s.pendingToggleLayerVisibility {
			actions.ToggleLayerVisibility = true
			s.pendingToggleLayerVisibility = false
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
		if s.pendingConvertPrefabName != nil {
			actions.ConvertSelectedEntityToPrefabName = *s.pendingConvertPrefabName
			actions.ApplyConvertSelectedEntityToPrefab = true
			s.pendingConvertPrefabName = nil
		}
		if s.pendingTransitionModeToggle {
			actions.ToggleTransitionMode = true
			s.pendingTransitionModeToggle = false
		}
		if s.pendingGateModeToggle {
			actions.ToggleGateMode = true
			s.pendingGateModeToggle = false
		}
		if s.pendingTransitionSelect != nil {
			actions.SelectEntity = *s.pendingTransitionSelect
			s.pendingTransitionSelect = nil
		}
		if s.pendingGateSelect != nil {
			actions.SelectEntity = *s.pendingGateSelect
			s.pendingGateSelect = nil
		}
		if s.pendingTransitionEdit != nil {
			actions.TransitionID = s.pendingTransitionEdit.ID
			actions.TransitionToLevel = s.pendingTransitionEdit.ToLevel
			actions.TransitionLinkedID = s.pendingTransitionEdit.LinkedID
			actions.TransitionEnterDir = s.pendingTransitionEdit.EnterDir
			actions.ApplyTransitionFields = true
			s.pendingTransitionEdit = nil
		}
		if s.pendingGateEdit != nil {
			actions.GateGroup = s.pendingGateEdit.Group
			actions.ApplyGateFields = true
			s.pendingGateEdit = nil
		}
		if s.pendingInspectorEdit != nil {
			actions.InspectorFieldComponent = s.pendingInspectorEdit.Component
			actions.InspectorFieldName = s.pendingInspectorEdit.Field
			actions.InspectorFieldValue = s.pendingInspectorEdit.Value
			actions.ApplyInspectorField = true
			s.pendingInspectorEdit = nil
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
