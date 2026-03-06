package editorsystem

import (
	"fmt"
	"math"
	"strings"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

type EditorAreaSystem struct {
	workspaceRoot string
}

func NewEditorAreaSystem(workspaceRoot string) *EditorAreaSystem {
	return &EditorAreaSystem{workspaceRoot: workspaceRoot}
}

func (s *EditorAreaSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok || session == nil {
		return
	}
	_, actions, _ := actionState(w)
	_, placement, _ := prefabPlacementState(w)
	_, selection, _ := entitySelectionState(w)
	_, drag, _ := areaDragState(w)
	_, overview, _ := overviewState(w)

	s.handleActions(w, session, actions, placement, selection, drag, overview)
	if session.OverviewOpen {
		return
	}
	if !session.TransitionMode && !session.GateMode {
		if drag != nil {
			drag.Active = false
			drag.EntityIndex = -1
			drag.Kind = ""
		}
		return
	}

	_, input, ok := rawInputState(w)
	if !ok || input == nil {
		return
	}
	_, pointer, ok := pointerState(w)
	if !ok || pointer == nil {
		return
	}
	_, meta, ok := levelMetaState(w)
	if !ok || meta == nil {
		return
	}
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}

	if placement != nil {
		placement.SelectedPath = ""
		placement.SelectedType = ""
	}
	if selection != nil {
		selection.Dragging = false
		selection.DragSnapshotDone = false
		selection.HoveredIndex = s.hoveredAreaIndex(pointer, entities.Items, session)
		if drag != nil && drag.PropertyEntityIndex != selection.SelectedIndex {
			selection.PropertySnapshotDone = false
			drag.PropertyEntityIndex = selection.SelectedIndex
		}
	}

	s.applyPropertyEdits(w, session, entities, selection, actions)

	if drag != nil && drag.Active {
		s.updateAreaDrag(w, session, meta, pointer, input, entities, selection, drag)
		return
	}

	if input.LeftJustPressed && pointer.InCanvas {
		if hovered := s.hoveredAreaIndex(pointer, entities.Items, session); hovered >= 0 {
			if selection != nil {
				selection.SelectedIndex = hovered
				selection.PropertySnapshotDone = false
			}
			s.beginResizeDrag(session, meta, entities, drag, hovered)
			return
		}
		if pointer.HasCell {
			s.beginCreateDrag(w, session, pointer, entities, selection, drag)
		}
	}
}

func (s *EditorAreaSystem) handleActions(w *ecs.World, session *editorcomponent.EditorSession, actions *editorcomponent.EditorActions, placement *editorcomponent.PrefabPlacementState, selection *editorcomponent.EntitySelectionState, drag *editorcomponent.AreaDragState, overview *editorcomponent.OverviewState) {
	if actions == nil || session == nil {
		return
	}
	if actions.ToggleTransitionMode {
		actions.ToggleTransitionMode = false
		session.TransitionMode = !session.TransitionMode
		if session.TransitionMode {
			session.GateMode = false
			session.OverviewOpen = false
			session.Status = "Transition mode enabled"
		} else {
			session.Status = "Transition mode disabled"
		}
		if selection != nil {
			selection.PropertySnapshotDone = false
		}
		resetTransientEditorState(w)
	}
	if actions.ToggleGateMode {
		actions.ToggleGateMode = false
		session.GateMode = !session.GateMode
		if session.GateMode {
			session.TransitionMode = false
			session.OverviewOpen = false
			session.Status = "Gate mode enabled"
		} else {
			session.Status = "Gate mode disabled"
		}
		if selection != nil {
			selection.PropertySnapshotDone = false
		}
		resetTransientEditorState(w)
	}
	if actions.ToggleOverview {
		actions.ToggleOverview = false
		session.OverviewOpen = !session.OverviewOpen
		if session.OverviewOpen {
			session.TransitionMode = false
			session.GateMode = false
			if overview != nil {
				overview.NeedsRefresh = true
			}
			session.Status = "Overview opened"
		} else {
			session.Status = "Overview closed"
		}
		resetTransientEditorState(w)
	}
	if actions.ClearSelections {
		session.TransitionMode = false
		session.GateMode = false
		if drag != nil {
			*drag = editorcomponent.AreaDragState{EntityIndex: -1, PropertyEntityIndex: -1}
		}
		if selection != nil {
			selection.PropertySnapshotDone = false
		}
	}
	if placement != nil && (session.TransitionMode || session.GateMode || session.OverviewOpen) {
		placement.SelectedPath = ""
		placement.SelectedType = ""
	}
}

func (s *EditorAreaSystem) hoveredAreaIndex(pointer *editorcomponent.PointerState, items []levels.Entity, session *editorcomponent.EditorSession) int {
	if pointer == nil || session == nil || !pointer.InCanvas {
		return -1
	}
	for index := len(items) - 1; index >= 0; index-- {
		item := items[index]
		if session.TransitionMode && !isTransitionEntity(item) {
			continue
		}
		if session.GateMode && !isGateEntity(item) {
			continue
		}
		left, top, width, height := entityRect(item)
		if pointer.WorldX >= left && pointer.WorldX <= left+width && pointer.WorldY >= top && pointer.WorldY <= top+height {
			return index
		}
	}
	return -1
}

func (s *EditorAreaSystem) beginCreateDrag(w *ecs.World, session *editorcomponent.EditorSession, pointer *editorcomponent.PointerState, entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState, drag *editorcomponent.AreaDragState) {
	if pointer == nil || entities == nil || drag == nil || session == nil {
		return
	}
	pushSnapshot(w, "area-create")
	item := levels.Entity{Type: "transition", X: pointer.CellX * TileSize, Y: pointer.CellY * TileSize, Props: map[string]interface{}{"w": float64(TileSize), "h": float64(TileSize), "layer": session.CurrentLayer}}
	kind := "transition"
	if session.GateMode {
		item.Type = "gate"
		item.Props["group"] = "boss_gate"
		kind = "gate"
	} else {
		item.ID = nextTransitionID(entities.Items)
		item.Props["id"] = item.ID
		item.Props["to_level"] = ""
		item.Props["linked_id"] = ""
		item.Props["enter_dir"] = "down"
	}
	entities.Items = append(entities.Items, item)
	index := len(entities.Items) - 1
	if selection != nil {
		selection.SelectedIndex = index
		selection.PropertySnapshotDone = false
	}
	*drag = editorcomponent.AreaDragState{Active: true, EntityIndex: index, Kind: kind, StartCellX: pointer.CellX, StartCellY: pointer.CellY, CurrentCellX: pointer.CellX, CurrentCellY: pointer.CellY, SnapshotDone: true, PropertyEntityIndex: index}
	session.Status = fmt.Sprintf("Created %s", kind)
	setDirty(w, true)
}

func (s *EditorAreaSystem) beginResizeDrag(session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, entities *editorcomponent.LevelEntities, drag *editorcomponent.AreaDragState, index int) {
	if drag == nil || entities == nil || index < 0 || index >= len(entities.Items) {
		return
	}
	item := entities.Items[index]
	left, top, width, height := entityRect(item)
	startCellX := clampInt(int(math.Floor(left/TileSize)), 0, maxInt(0, meta.Width-1))
	startCellY := clampInt(int(math.Floor(top/TileSize)), 0, maxInt(0, meta.Height-1))
	endCellX := clampInt(int(math.Floor((left+width-1)/TileSize)), 0, maxInt(0, meta.Width-1))
	endCellY := clampInt(int(math.Floor((top+height-1)/TileSize)), 0, maxInt(0, meta.Height-1))
	kind := "gate"
	if session != nil && session.TransitionMode {
		kind = "transition"
	}
	*drag = editorcomponent.AreaDragState{Active: true, EntityIndex: index, Kind: kind, StartCellX: startCellX, StartCellY: startCellY, CurrentCellX: endCellX, CurrentCellY: endCellY, PropertyEntityIndex: index}
}

func (s *EditorAreaSystem) updateAreaDrag(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState, drag *editorcomponent.AreaDragState) {
	if drag == nil || entities == nil || drag.EntityIndex < 0 || drag.EntityIndex >= len(entities.Items) {
		if drag != nil {
			drag.Active = false
		}
		return
	}
	if !input.LeftDown && !input.LeftJustReleased {
		drag.Active = false
		return
	}
	if pointer != nil && pointer.InCanvas {
		drag.CurrentCellX, drag.CurrentCellY = clampedPointerCell(meta, pointer)
		if !drag.SnapshotDone {
			pushSnapshot(w, "area-resize")
			drag.SnapshotDone = true
		}
		s.applyAreaRect(meta, &entities.Items[drag.EntityIndex], drag.StartCellX, drag.StartCellY, drag.CurrentCellX, drag.CurrentCellY, session.CurrentLayer)
		setDirty(w, true)
	}
	if input.LeftJustReleased || !input.LeftDown {
		drag.Active = false
		if selection != nil && drag.EntityIndex >= 0 {
			selection.SelectedIndex = drag.EntityIndex
		}
		session.Status = fmt.Sprintf("Updated %s", drag.Kind)
	}
}

func (s *EditorAreaSystem) applyPropertyEdits(w *ecs.World, session *editorcomponent.EditorSession, entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState, actions *editorcomponent.EditorActions) {
	if actions == nil || entities == nil || selection == nil || selection.SelectedIndex < 0 || selection.SelectedIndex >= len(entities.Items) {
		return
	}
	item := &entities.Items[selection.SelectedIndex]
	if actions.ApplyTransitionFields && isTransitionEntity(*item) {
		if !selection.PropertySnapshotDone {
			pushSnapshot(w, "transition-props")
			selection.PropertySnapshotDone = true
		}
		item.ID = strings.TrimSpace(actions.TransitionID)
		props := ensureEntityProps(item)
		props["id"] = item.ID
		props["to_level"] = strings.TrimSpace(actions.TransitionToLevel)
		if strings.TrimSpace(actions.TransitionToLevel) != "" {
			props["to_level"] = normalizeLevelRef(actions.TransitionToLevel)
		}
		props["linked_id"] = strings.TrimSpace(actions.TransitionLinkedID)
		direction := strings.ToLower(strings.TrimSpace(actions.TransitionEnterDir))
		if direction == "up" || direction == "down" || direction == "left" || direction == "right" {
			props["enter_dir"] = direction
		}
		setDirty(w, true)
		session.Status = "Updated transition properties"
		actions.ApplyTransitionFields = false
	}
	if actions.ApplyGateFields && isGateEntity(*item) {
		if !selection.PropertySnapshotDone {
			pushSnapshot(w, "gate-props")
			selection.PropertySnapshotDone = true
		}
		props := ensureEntityProps(item)
		group := strings.TrimSpace(actions.GateGroup)
		if group == "" {
			group = "boss_gate"
		}
		props["group"] = group
		setDirty(w, true)
		session.Status = "Updated gate properties"
		actions.ApplyGateFields = false
	}
}

func (s *EditorAreaSystem) applyAreaRect(meta *editorcomponent.LevelMeta, item *levels.Entity, startX, startY, endX, endY, layerIndex int) {
	if meta == nil || item == nil {
		return
	}
	left := minInt(startX, endX)
	top := minInt(startY, endY)
	right := maxInt(startX, endX)
	bottom := maxInt(startY, endY)
	item.X = clampInt(left, 0, maxInt(0, meta.Width-1)) * TileSize
	item.Y = clampInt(top, 0, maxInt(0, meta.Height-1)) * TileSize
	props := ensureEntityProps(item)
	props["w"] = float64((right - left + 1) * TileSize)
	props["h"] = float64((bottom - top + 1) * TileSize)
	props["layer"] = layerIndex
}

func clampedPointerCell(meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState) (int, int) {
	if meta == nil || pointer == nil {
		return 0, 0
	}
	return clampInt(pointer.CellX, 0, maxInt(0, meta.Width-1)), clampInt(pointer.CellY, 0, maxInt(0, meta.Height-1))
}

func nextTransitionID(items []levels.Entity) string {
	used := make(map[string]struct{}, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = entityStringProp(item, "id")
		}
		if id != "" {
			used[id] = struct{}{}
		}
	}
	for index := 1; ; index++ {
		candidate := fmt.Sprintf("t%d", index)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

var _ ecs.System = (*EditorAreaSystem)(nil)
