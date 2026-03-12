package editorsystem

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	editorui "github.com/milk9111/sidescroller/cmd/editor/ui"
	editoruicomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

type EditorUISystem struct {
	ui                             *editorui.EditorUI
	syncedEntitySelection          int
	cachedInspectorValid           bool
	cachedInspectorSelection       int
	cachedInspectorEntity          levels.Entity
	cachedInspectorPrefab          *editorio.PrefabInfo
	cachedInspectorState           editoruicomponents.InspectorState
	inspectorFeedbackSelection     int
	inspectorFeedbackStatus        string
	inspectorFeedbackParseError    string
	inspectorApplyPending          bool
	pendingTool                    *editorcomponent.ToolKind
	pendingAsset                   *editorio.AssetInfo
	pendingTileSelection           *model.TileSelection
	pendingPrefab                  *editorio.PrefabInfo
	pendingSaveTarget              *string
	pendingSave                    bool
	pendingLayerSelect             *int
	pendingLayerDeleteArm          *bool
	pendingLayerAdd                bool
	pendingLayerMove               int
	pendingLayerRename             *string
	pendingTogglePhysics           bool
	pendingToggleLayerVisibility   bool
	pendingTogglePhysicsHighlight  bool
	pendingToggleAutotile          bool
	pendingEntitySelect            *int
	pendingConvertPrefabName       *string
	pendingTransitionModeToggle    bool
	pendingGateModeToggle          bool
	pendingTransitionSelect        *int
	pendingGateSelect              *int
	transitionDraftSelection       int
	transitionDraft                *editoruicomponents.TransitionEditorState
	pendingTransitionEditSelection int
	pendingTransitionEdit          *editoruicomponents.TransitionEditorState
	pendingGateEdit                *editoruicomponents.GateEditorState
	pendingInspectorDocument       *string
	pendingResizeWidth             string
	pendingResizeHeight            string
	pendingResize                  bool
}

func NewEditorUISystem(assets []editorio.AssetInfo, prefabs []editorio.PrefabInfo) (*EditorUISystem, error) {
	system := &EditorUISystem{syncedEntitySelection: -1, transitionDraftSelection: -1, pendingTransitionEditSelection: -1, inspectorFeedbackSelection: -1}
	setLayerDeleteArm := func(armed bool) {
		copied := armed
		system.pendingLayerDeleteArm = &copied
	}
	_ = prefabs
	ui, err := editorui.NewEditorUI(assets, editorui.Callbacks{
		OnToolSelected: func(tool editorcomponent.ToolKind) {
			system.pendingTool = &tool
			setLayerDeleteArm(false)
		},
		OnAssetSelected: func(asset editorio.AssetInfo) {
			copied := asset
			system.pendingAsset = &copied
			setLayerDeleteArm(false)
		},
		OnTileSelected: func(selection model.TileSelection) {
			copied := selection.Normalize()
			system.pendingTileSelection = &copied
			setLayerDeleteArm(false)
		},
		OnPrefabSelected: func(prefab editorio.PrefabInfo) {
			copied := prefab
			system.pendingPrefab = &copied
			setLayerDeleteArm(false)
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
			setLayerDeleteArm(true)
		},
		OnLayerAdded: func() {
			system.pendingLayerAdd = true
			setLayerDeleteArm(true)
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
			setLayerDeleteArm(false)
		},
		OnConvertToPrefabConfirmed: func(name string) {
			copied := name
			system.pendingConvertPrefabName = &copied
			setLayerDeleteArm(false)
		},
		OnTransitionModeToggled: func() {
			system.pendingTransitionModeToggle = true
			setLayerDeleteArm(false)
		},
		OnGateModeToggled: func() {
			system.pendingGateModeToggle = true
			setLayerDeleteArm(false)
		},
		OnTransitionSelected: func(index int) {
			copied := index
			system.pendingTransitionSelect = &copied
			system.clearTransitionDraft()
			setLayerDeleteArm(false)
		},
		OnGateSelected: func(index int) {
			copied := index
			system.pendingGateSelect = &copied
			setLayerDeleteArm(false)
		},
		OnTransitionEdited: func(state editoruicomponents.TransitionEditorState) {
			copied := state
			system.transitionDraft = &copied
			system.transitionDraftSelection = system.currentTransitionSelection()
			system.pendingTransitionEditSelection = system.currentTransitionSelection()
			system.pendingTransitionEdit = &copied
		},
		OnGateEdited: func(state editoruicomponents.GateEditorState) {
			copied := state
			system.pendingGateEdit = &copied
		},
		OnInspectorDocumentSaved: func(document string) {
			copied := document
			system.pendingInspectorDocument = &copied
		},
		OnLevelResizeRequested: func(widthText, heightText string) {
			system.pendingResizeWidth = widthText
			system.pendingResizeHeight = heightText
			system.pendingResize = true
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
	selectedEntity := -1
	if entitySelection != nil {
		selectedEntity = entitySelection.SelectedIndex
	}
	s.syncInspectorFeedback(session, selectedEntity)
	inspectorState := s.inspectorStateForSelection(prefabCatalog, levelEntities, selectedEntity)
	inspectorState = s.decorateInspectorState(inspectorState, selectedEntity)
	s.syncedEntitySelection = selectedEntity
	currentTransitionSelection := currentTransitionSelectionIndex(entitySelection, s.pendingTransitionSelect, levelEntities)
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
		transitionState = s.mergeTransitionDraft(transitionState, currentTransitionSelection)
	}
	s.ui.Sync(session.ActiveTool, session.SaveTarget, width, height, session.CurrentLayer, len(layerEntities(w)), layers, autotileEnabled, session.PhysicsHighlight, session.Dirty, prefabItems, selectedPrefabPath, entityItems, selectedEntity, session.TransitionMode, session.GateMode, transitionItems, gateItems, transitionState, gateState, session.SelectedTile.Path, selectedIndex, session.Status, inspectorState)
	s.ui.Update()
	s.captureTransitionDraftFromUI(currentTransitionSelectionIndex(entitySelection, s.pendingTransitionSelect, levelEntities))
	s.syncTransitionDraftToWorld(w, session, entitySelection, levelEntities)
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
		if s.pendingLayerDeleteArm != nil {
			focus.LayerDeleteArmed = *s.pendingLayerDeleteArm
			s.pendingLayerDeleteArm = nil
		}
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
	if actions != nil {
		if s.pendingResize {
			width, widthErr := strconv.Atoi(strings.TrimSpace(s.pendingResizeWidth))
			height, heightErr := strconv.Atoi(strings.TrimSpace(s.pendingResizeHeight))
			s.pendingResize = false
			if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
				session.Status = "Level size must be positive integers"
			} else {
				actions.ResizeWidth = width
				actions.ResizeHeight = height
				actions.ApplyLevelResize = true
			}
		}
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
			switch s.transitionEditDispatchState(entitySelection, levelEntities) {
			case transitionEditReady:
				actions.TransitionID = s.pendingTransitionEdit.ID
				actions.TransitionToLevel = s.pendingTransitionEdit.ToLevel
				actions.TransitionLinkedID = s.pendingTransitionEdit.LinkedID
				actions.TransitionEnterDir = s.pendingTransitionEdit.EnterDir
				actions.ApplyTransitionFields = true
				s.pendingTransitionEdit = nil
				s.pendingTransitionEditSelection = -1
			case transitionEditStale:
				s.pendingTransitionEdit = nil
				s.pendingTransitionEditSelection = -1
			}
		}
		if s.pendingGateEdit != nil {
			actions.GateGroup = s.pendingGateEdit.Group
			actions.ApplyGateFields = true
			s.pendingGateEdit = nil
		}
		s.dispatchPendingInspectorDocument(actions, selectedEntity)
	}
	if s.pendingSave {
		s.flushTransitionDraftToEntities(w, levelEntities)
		if s.pendingTransitionEdit == nil {
			session.SaveRequested = true
			s.pendingSave = false
		} else if session.Status == "Ready" || session.Status == "Saved levels/"+session.LoadedLevel {
			session.Status = "Waiting for transition edits"
		}
	}
}

type transitionEditDispatch int

const (
	transitionEditWait transitionEditDispatch = iota
	transitionEditReady
	transitionEditStale
)

func (s *EditorUISystem) transitionEditDispatchState(selection *editorcomponent.EntitySelectionState, entities *editorcomponent.LevelEntities) transitionEditDispatch {
	if s == nil || s.pendingTransitionEdit == nil {
		return transitionEditStale
	}
	if entities == nil {
		return transitionEditWait
	}
	if pendingTransitionSelectionMatches(s.pendingTransitionEditSelection, s.pendingTransitionSelect) {
		return transitionEditWait
	}
	if selection == nil || selection.SelectedIndex != s.pendingTransitionEditSelection {
		return transitionEditStale
	}
	if selection.SelectedIndex < 0 || selection.SelectedIndex >= len(entities.Items) || !isTransitionEntity(entities.Items[selection.SelectedIndex]) {
		return transitionEditStale
	}
	return transitionEditReady
}

func (s *EditorUISystem) dispatchPendingInspectorDocument(actions *editorcomponent.EditorActions, selectedIndex int) {
	if s == nil || actions == nil || s.pendingInspectorDocument == nil {
		return
	}
	actions.InspectorDocument = *s.pendingInspectorDocument
	actions.ApplyInspectorDocument = true
	s.noteInspectorDocumentDispatch(selectedIndex)
	s.pendingInspectorDocument = nil
}

func (s *EditorUISystem) noteInspectorDocumentDispatch(selectedIndex int) {
	if s == nil {
		return
	}
	s.inspectorFeedbackSelection = selectedIndex
	s.inspectorFeedbackStatus = ""
	s.inspectorFeedbackParseError = ""
	s.inspectorApplyPending = true
}

func (s *EditorUISystem) syncInspectorFeedback(session *editorcomponent.EditorSession, selectedIndex int) {
	if s == nil {
		return
	}
	if !s.inspectorApplyPending && s.inspectorFeedbackSelection != -1 && s.inspectorFeedbackSelection != selectedIndex {
		s.clearInspectorFeedback()
	}
	if session == nil || !s.inspectorApplyPending {
		return
	}
	status := strings.TrimSpace(session.Status)
	switch {
	case strings.HasPrefix(status, "Inspector apply failed: "):
		s.inspectorFeedbackStatus = "Inspector apply failed"
		s.inspectorFeedbackParseError = strings.TrimSpace(strings.TrimPrefix(status, "Inspector apply failed: "))
		s.inspectorApplyPending = false
	case isInspectorTerminalStatus(status):
		s.inspectorFeedbackStatus = status
		s.inspectorFeedbackParseError = ""
		s.inspectorApplyPending = false
	}
}

func (s *EditorUISystem) decorateInspectorState(state editoruicomponents.InspectorState, selectedIndex int) editoruicomponents.InspectorState {
	if s == nil {
		return state
	}
	state.Dirty = s.currentInspectorEditorDirty(selectedIndex)
	if state.Active && strings.TrimSpace(state.DocumentText) != "" && strings.TrimSpace(state.StatusMessage) == "" {
		state.StatusMessage = "Edit component YAML and press Ctrl+S to apply"
	}
	if s.inspectorFeedbackSelection != selectedIndex {
		return state
	}
	if strings.TrimSpace(s.inspectorFeedbackStatus) != "" {
		state.StatusMessage = s.inspectorFeedbackStatus
	}
	if strings.TrimSpace(s.inspectorFeedbackParseError) != "" {
		state.ParseError = s.inspectorFeedbackParseError
		return state
	}
	if isInspectorSuccessfulStatus(s.inspectorFeedbackStatus) {
		state.Dirty = false
	}
	return state
}

func (s *EditorUISystem) currentInspectorEditorDirty(selectedIndex int) bool {
	if s == nil || s.ui == nil || selectedIndex < 0 || s.ui.AssetPanel == nil || s.ui.AssetPanel.Inspector == nil || s.ui.AssetPanel.Inspector.Editor == nil {
		return false
	}
	return s.ui.AssetPanel.Inspector.Editor.IsDirty()
}

func (s *EditorUISystem) clearInspectorFeedback() {
	if s == nil {
		return
	}
	s.inspectorFeedbackSelection = -1
	s.inspectorFeedbackStatus = ""
	s.inspectorFeedbackParseError = ""
	s.inspectorApplyPending = false
}

func isInspectorTerminalStatus(status string) bool {
	return isInspectorSuccessfulStatus(status) || status == "Select an entity to inspect"
}

func isInspectorSuccessfulStatus(status string) bool {
	return status == "Updated entity component overrides" || status == "Inspector already up to date"
}

func pendingTransitionSelectionMatches(index int, pendingSelect *int) bool {
	if pendingSelect == nil {
		return false
	}
	return *pendingSelect == index
}

func (s *EditorUISystem) captureTransitionDraftFromUI(selectionIndex int) {
	if s == nil || s.ui == nil || selectionIndex < 0 {
		return
	}
	state, ok := s.ui.CurrentTransitionEditorState()
	if !ok {
		return
	}
	copied := state
	s.transitionDraftSelection = selectionIndex
	s.transitionDraft = &copied
	s.pendingTransitionEditSelection = selectionIndex
	s.pendingTransitionEdit = &copied
}

func (s *EditorUISystem) syncTransitionDraftToWorld(w *ecs.World, session *editorcomponent.EditorSession, selection *editorcomponent.EntitySelectionState, entities *editorcomponent.LevelEntities) bool {
	if s == nil || w == nil || s.transitionDraft == nil || entities == nil {
		return false
	}
	index := s.transitionDraftSelection
	if index < 0 || index >= len(entities.Items) {
		return false
	}
	item := &entities.Items[index]
	if !isTransitionEntity(*item) {
		return false
	}
	current := editorStateForTransitionEntity(*item)
	if transitionEditorStatesEqual(current, *s.transitionDraft) {
		s.pendingTransitionEdit = nil
		s.pendingTransitionEditSelection = -1
		return false
	}
	if selection != nil && !selection.PropertySnapshotDone {
		pushSnapshot(w, "transition-props")
		selection.PropertySnapshotDone = true
	}
	changed := applyTransitionEditorState(item, *s.transitionDraft)
	if changed {
		setDirty(w, true)
		if session != nil {
			session.Status = "Updated transition properties"
		}
	}
	s.pendingTransitionEdit = nil
	s.pendingTransitionEditSelection = -1
	return changed
}

func editorStateForTransitionEntity(item levels.Entity) editoruicomponents.TransitionEditorState {
	state := editoruicomponents.TransitionEditorState{
		Selected: isTransitionEntity(item),
		ID:       entityStringProp(item, "id"),
		ToLevel:  entityStringProp(item, "to_level"),
		LinkedID: entityStringProp(item, "linked_id"),
		EnterDir: entityStringProp(item, "enter_dir"),
	}
	if state.ID == "" {
		state.ID = strings.TrimSpace(item.ID)
	}
	if state.EnterDir == "" {
		state.EnterDir = "down"
	}
	return state
}

func (s *EditorUISystem) flushTransitionDraftToEntities(w *ecs.World, entities *editorcomponent.LevelEntities) bool {
	if s == nil || w == nil || s.transitionDraft == nil || entities == nil {
		return false
	}
	index := s.transitionDraftSelection
	if index < 0 || index >= len(entities.Items) {
		return false
	}
	item := &entities.Items[index]
	if !isTransitionEntity(*item) {
		return false
	}
	if applyTransitionEditorState(item, *s.transitionDraft) {
		setDirty(w, true)
		return true
	}
	return false
}

func applyTransitionEditorState(item *levels.Entity, state editoruicomponents.TransitionEditorState) bool {
	if item == nil {
		return false
	}
	changed := false
	id := strings.TrimSpace(state.ID)
	if item.ID != id {
		item.ID = id
		changed = true
	}
	props := ensureEntityProps(item)
	if value, _ := props["id"].(string); value != id {
		props["id"] = id
		changed = true
	}
	toLevel := strings.TrimSpace(state.ToLevel)
	if toLevel != "" {
		toLevel = normalizeLevelRef(toLevel)
	}
	if value, _ := props["to_level"].(string); value != toLevel {
		props["to_level"] = toLevel
		changed = true
	}
	linkedID := strings.TrimSpace(state.LinkedID)
	if value, _ := props["linked_id"].(string); value != linkedID {
		props["linked_id"] = linkedID
		changed = true
	}
	direction := strings.ToLower(strings.TrimSpace(state.EnterDir))
	if direction == "up" || direction == "down" || direction == "left" || direction == "right" {
		if value, _ := props["enter_dir"].(string); value != direction {
			props["enter_dir"] = direction
			changed = true
		}
	}
	return changed
}

func currentTransitionSelectionIndex(selection *editorcomponent.EntitySelectionState, pendingSelect *int, entities *editorcomponent.LevelEntities) int {
	if entities == nil {
		return -1
	}
	if pendingSelect != nil {
		idx := *pendingSelect
		if idx >= 0 && idx < len(entities.Items) && isTransitionEntity(entities.Items[idx]) {
			return idx
		}
		return -1
	}
	if selection == nil {
		return -1
	}
	idx := selection.SelectedIndex
	if idx >= 0 && idx < len(entities.Items) && isTransitionEntity(entities.Items[idx]) {
		return idx
	}
	return -1
}

func (s *EditorUISystem) currentTransitionSelection() int {
	if s == nil {
		return -1
	}
	if s.pendingTransitionSelect != nil {
		return *s.pendingTransitionSelect
	}
	return s.syncedEntitySelection
}

func (s *EditorUISystem) clearTransitionDraft() {
	if s == nil {
		return
	}
	s.transitionDraft = nil
	s.transitionDraftSelection = -1
}

func (s *EditorUISystem) mergeTransitionDraft(base editoruicomponents.TransitionEditorState, selectionIndex int) editoruicomponents.TransitionEditorState {
	if s == nil || s.transitionDraft == nil {
		if base.EnterDir == "" {
			base.EnterDir = "down"
		}
		return base
	}
	if selectionIndex != s.transitionDraftSelection || selectionIndex < 0 || !base.Selected {
		s.clearTransitionDraft()
		if base.EnterDir == "" {
			base.EnterDir = "down"
		}
		return base
	}
	if transitionEditorStatesEqual(base, *s.transitionDraft) {
		s.clearTransitionDraft()
		if base.EnterDir == "" {
			base.EnterDir = "down"
		}
		return base
	}
	merged := *s.transitionDraft
	merged.Selected = true
	if merged.EnterDir == "" {
		merged.EnterDir = base.EnterDir
	}
	if merged.EnterDir == "" {
		merged.EnterDir = "down"
	}
	return merged
}

func transitionEditorStatesEqual(left, right editoruicomponents.TransitionEditorState) bool {
	return left.Selected == right.Selected &&
		left.ID == right.ID &&
		left.ToLevel == right.ToLevel &&
		left.LinkedID == right.LinkedID &&
		left.EnterDir == right.EnterDir
}

func (s *EditorUISystem) inspectorStateForSelection(catalog *editorcomponent.PrefabCatalog, entities *editorcomponent.LevelEntities, selectedIndex int) editoruicomponents.InspectorState {
	item, prefab, ok := inspectorCacheInputs(catalog, entities, selectedIndex)
	if s.cachedInspectorValid && s.cachedInspectorSelection == selectedIndex {
		if !ok {
			return s.cachedInspectorState
		}
		if reflect.DeepEqual(s.cachedInspectorEntity, item) && reflect.DeepEqual(s.cachedInspectorPrefab, prefab) {
			return s.cachedInspectorState
		}
	}
	state := buildInspectorState(catalog, entities, selectedIndex)
	s.cachedInspectorValid = true
	s.cachedInspectorSelection = selectedIndex
	s.cachedInspectorState = state
	if !ok {
		s.cachedInspectorEntity = levels.Entity{}
		s.cachedInspectorPrefab = nil
		return state
	}
	s.cachedInspectorEntity = cloneEditorEntity(item)
	s.cachedInspectorPrefab = cloneInspectorPrefab(prefab)
	return state
}

func inspectorCacheInputs(catalog *editorcomponent.PrefabCatalog, entities *editorcomponent.LevelEntities, selectedIndex int) (levels.Entity, *editorio.PrefabInfo, bool) {
	if entities == nil || selectedIndex < 0 || selectedIndex >= len(entities.Items) {
		return levels.Entity{}, nil, false
	}
	item := entities.Items[selectedIndex]
	return item, prefabInfoForEntity(catalog, item), true
}

func cloneInspectorPrefab(prefab *editorio.PrefabInfo) *editorio.PrefabInfo {
	if prefab == nil {
		return nil
	}
	cloned := *prefab
	cloned.Components = editorio.MergeComponentMaps(prefab.Components, nil)
	return &cloned
}

func (s *EditorUISystem) Draw(screen *ebiten.Image) {
	if s == nil || s.ui == nil {
		return
	}
	s.ui.Draw(screen)
}

var _ ecs.System = (*EditorUISystem)(nil)
