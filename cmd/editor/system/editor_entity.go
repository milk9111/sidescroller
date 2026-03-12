package editorsystem

import (
	"fmt"
	"sort"
	"strings"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

type EditorEntitySystem struct{}

func NewEditorEntitySystem() *EditorEntitySystem {
	return &EditorEntitySystem{}
}

func (s *EditorEntitySystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok || session == nil {
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
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}
	_, prefabs, _ := prefabCatalogState(w)
	_, placement, ok := prefabPlacementState(w)
	if !ok || placement == nil {
		return
	}
	_, clipboard, _ := clipboardState(w)
	_, selection, ok := entitySelectionState(w)
	if !ok || selection == nil {
		return
	}
	_, moveSelection, _ := moveSelectionState(w)
	_, actions, _ := actionState(w)

	s.clampSelection(w, session, entities, selection)
	selection.HoveredIndex = s.hoveredEntityIndex(w, session, pointer, entities.Items, prefabs)

	if actions != nil {
		if actions.CopySelectedEntity {
			actions.CopySelectedEntity = false
			if clipboard != nil && selection.SelectedIndex >= 0 && selection.SelectedIndex < len(entities.Items) {
				clipboard.Entity = cloneEditorEntity(entities.Items[selection.SelectedIndex])
				clipboard.Valid = true
				session.Status = "Copied entity"
			} else {
				session.Status = "Select an entity to copy"
			}
		}
		if actions.PasteCopiedEntity {
			actions.PasteCopiedEntity = false
			if clipboard == nil || !clipboard.Valid {
				session.Status = "Copy an entity first"
			} else if s.pasteCopiedEntity(w, session, entities, placement, selection, clipboard) {
				session.Status = "Pasted entity"
			}
		}
		if strings.TrimSpace(actions.SelectPrefab) != "" {
			if info := prefabInfoByPath(prefabs, actions.SelectPrefab, ""); info != nil {
				placement.SelectedPath = info.Path
				placement.SelectedType = info.EntityType
				selection.SelectedIndex = -1
				s.clearEntityDrag(selection)
				session.Status = "Selected prefab " + info.Name
			}
			actions.SelectPrefab = ""
		}
		if actions.SelectEntity >= 0 {
			if actions.SelectEntity < len(entities.Items) {
				if entitySelectableOnCurrentLayer(w, session, entities.Items[actions.SelectEntity]) && selection.SelectedIndex != actions.SelectEntity {
					selection.SelectedIndex = actions.SelectEntity
					s.clearEntityDrag(selection)
				}
				if entitySelectableOnCurrentLayer(w, session, entities.Items[actions.SelectEntity]) {
					placement.SelectedPath = ""
					placement.SelectedType = ""
					session.Status = "Selected entity"
				}
			}
			actions.SelectEntity = -1
		}
		if actions.DeleteSelectedEntity {
			actions.DeleteSelectedEntity = false
			if s.deleteSelectedEntity(w, entities, selection) {
				placement.SelectedPath = ""
				placement.SelectedType = ""
				session.Status = "Deleted entity"
			}
		}
		if actions.ClearSelections {
			actions.ClearSelections = false
			placement.SelectedPath = ""
			placement.SelectedType = ""
			selection.SelectedIndex = -1
			selection.HoveredIndex = -1
			s.clearEntityDrag(selection)
			if moveSelection != nil {
				*moveSelection = editorcomponent.MoveSelectionState{}
			}
			session.Status = "Cleared selection"
		}
		if actions.ApplyInspectorField {
			fmt.Println("Applying inspector field edit:", actions.InspectorFieldComponent, actions.InspectorFieldName, actions.InspectorFieldValue)
			if selection.SelectedIndex >= 0 && selection.SelectedIndex < len(entities.Items) {
				selected := &entities.Items[selection.SelectedIndex]
				if prefab := prefabInfoForEntity(prefabs, *selected); prefab != nil {
					if !selection.PropertySnapshotDone {
						pushSnapshot(w, "entity-inspector")
						selection.PropertySnapshotDone = true
					}
					if applyInspectorFieldEdit(selected, prefab, actions.InspectorFieldComponent, actions.InspectorFieldName, actions.InspectorFieldValue) {
						setDirty(w, true)
						session.Status = "Updated entity component"
					}
				}
			}
			actions.ApplyInspectorField = false
		}
		if actions.ApplyInspectorDocument {
			actions.InspectorDocument = strings.ReplaceAll(actions.InspectorDocument, "\r\n", "\n")
			actions.ApplyInspectorDocument = false
			if selection.SelectedIndex < 0 || selection.SelectedIndex >= len(entities.Items) {
				session.Status = "Select an entity to inspect"
				actions.InspectorDocument = ""
			} else {
				selected := &entities.Items[selection.SelectedIndex]
				prefab := prefabInfoForEntity(prefabs, *selected)
				updated := cloneEditorEntity(*selected)
				changed, err := applyInspectorDocumentEdit(&updated, prefab, actions.InspectorDocument)
				switch {
				case err != nil:
					session.Status = fmt.Sprintf("Inspector apply failed: %v", err)
				case changed:
					if !selection.PropertySnapshotDone {
						pushSnapshot(w, "entity-inspector")
					}
					*selected = updated
					selection.PropertySnapshotDone = false
					setDirty(w, true)
					session.Status = "Updated entity component overrides"
				default:
					selection.PropertySnapshotDone = false
					session.Status = "Inspector already up to date"
				}
				actions.InspectorDocument = ""
			}
		}
	}

	if session.OverviewOpen || session.TransitionMode || session.GateMode {
		s.clearEntityDrag(selection)
		return
	}
	if session.ActiveTool == editorcomponent.ToolMove {
		selection.HoveredIndex = -1
		s.clearEntityDrag(selection)
		return
	}

	if input.LeftJustPressed && pointer.InCanvas {
		clearLayerDeleteArm(w)
	}

	if placement.SelectedPath != "" {
		if input.LeftJustPressed && pointer.HasCell {
			if s.placePrefab(w, session, entities, placement, pointer) {
				selection.SelectedIndex = len(entities.Items) - 1
				s.clearEntityDrag(selection)
			}
		}
		return
	}

	if selection.Dragging {
		s.updateEntityDrag(w, pointer, input, entities, selection)
		return
	}

	if input.LeftJustPressed && pointer.InCanvas {
		if selection.HoveredIndex >= 0 {
			selection.SelectedIndex = selection.HoveredIndex
			s.beginEntityDrag(pointer, selection, entities.Items[selection.SelectedIndex])
			session.Status = "Selected entity"
			return
		}
	}
}

func (s *EditorEntitySystem) pasteCopiedEntity(w *ecs.World, session *editorcomponent.EditorSession, entities *editorcomponent.LevelEntities, placement *editorcomponent.PrefabPlacementState, selection *editorcomponent.EntitySelectionState, clipboard *editorcomponent.EntityClipboardState) bool {
	if entities == nil || selection == nil || clipboard == nil || !clipboard.Valid {
		return false
	}
	pushSnapshot(w, "entity-paste")
	copy := cloneEditorEntity(clipboard.Entity)
	entities.Items = append(entities.Items, copy)
	ensureUniqueEntityIDs(entities.Items)
	selection.SelectedIndex = len(entities.Items) - 1
	selection.HoveredIndex = -1
	selection.PropertySnapshotDone = false
	placement.SelectedPath = ""
	placement.SelectedType = ""
	s.clearEntityDrag(selection)
	setDirty(w, true)
	_ = session
	return true
}

func (s *EditorEntitySystem) clampSelection(w *ecs.World, session *editorcomponent.EditorSession, entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState) {
	if entities == nil || selection == nil {
		return
	}
	if len(entities.Items) == 0 {
		selection.SelectedIndex = -1
		selection.HoveredIndex = -1
		selection.Dragging = false
		selection.DragOffsetCellX = 0
		selection.DragOffsetCellY = 0
		selection.DragSnapshotDone = false
		return
	}
	if selection.SelectedIndex >= len(entities.Items) {
		selection.SelectedIndex = len(entities.Items) - 1
	}
	if selection.SelectedIndex < -1 {
		selection.SelectedIndex = -1
	}
	if selection.SelectedIndex >= 0 && selection.SelectedIndex < len(entities.Items) && !entitySelectableOnCurrentLayer(w, session, entities.Items[selection.SelectedIndex]) {
		selection.SelectedIndex = -1
		selection.HoveredIndex = -1
		selection.Dragging = false
		selection.DragOffsetCellX = 0
		selection.DragOffsetCellY = 0
		selection.DragSnapshotDone = false
	}
}

func (s *EditorEntitySystem) placePrefab(w *ecs.World, session *editorcomponent.EditorSession, entities *editorcomponent.LevelEntities, placement *editorcomponent.PrefabPlacementState, pointer *editorcomponent.PointerState) bool {
	if entities == nil || placement == nil || pointer == nil || placement.SelectedType == "" {
		return false
	}
	pushSnapshot(w, "entity-place")
	props := map[string]interface{}{
		"layer":  session.CurrentLayer,
		"prefab": placement.SelectedPath,
	}
	entities.Items = append(entities.Items, levels.Entity{
		Type:  placement.SelectedType,
		X:     pointer.CellX * TileSize,
		Y:     pointer.CellY * TileSize,
		Props: props,
	})
	setDirty(w, true)
	session.Status = "Placed entity " + placement.SelectedType
	return true
}

func (s *EditorEntitySystem) deleteSelectedEntity(w *ecs.World, entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState) bool {
	if entities == nil || selection == nil || selection.SelectedIndex < 0 || selection.SelectedIndex >= len(entities.Items) {
		return false
	}
	pushSnapshot(w, "entity-delete")
	entities.Items = append(entities.Items[:selection.SelectedIndex], entities.Items[selection.SelectedIndex+1:]...)
	if selection.SelectedIndex >= len(entities.Items) {
		selection.SelectedIndex = len(entities.Items) - 1
	}
	s.clearEntityDrag(selection)
	setDirty(w, true)
	return true
}

func (s *EditorEntitySystem) updateEntityDrag(w *ecs.World, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState) {
	if entities == nil || selection == nil || selection.SelectedIndex < 0 || selection.SelectedIndex >= len(entities.Items) {
		s.clearEntityDrag(selection)
		return
	}
	if input.LeftJustReleased || !input.LeftDown || !pointer.InCanvas || !pointer.HasCell {
		s.clearEntityDrag(selection)
		return
	}
	if !selection.DragSnapshotDone {
		pushSnapshot(w, "entity-drag")
		selection.DragSnapshotDone = true
	}
	item := &entities.Items[selection.SelectedIndex]
	nextCellX := pointer.CellX - selection.DragOffsetCellX
	nextCellY := pointer.CellY - selection.DragOffsetCellY
	if nextCellX < 0 {
		nextCellX = 0
	}
	if nextCellY < 0 {
		nextCellY = 0
	}
	nextX := nextCellX * TileSize
	nextY := nextCellY * TileSize
	if item.X != nextX || item.Y != nextY {
		item.X = nextX
		item.Y = nextY
		setDirty(w, true)
	}
}

func (s *EditorEntitySystem) beginEntityDrag(pointer *editorcomponent.PointerState, selection *editorcomponent.EntitySelectionState, item levels.Entity) {
	if selection == nil {
		return
	}
	selection.Dragging = s.entityDraggable(item)
	selection.DragOffsetCellX = 0
	selection.DragOffsetCellY = 0
	selection.DragSnapshotDone = false
	if !selection.Dragging || pointer == nil || !pointer.HasCell {
		return
	}
	selection.DragOffsetCellX = pointer.CellX - item.X/TileSize
	selection.DragOffsetCellY = pointer.CellY - item.Y/TileSize
}

func (s *EditorEntitySystem) clearEntityDrag(selection *editorcomponent.EntitySelectionState) {
	if selection == nil {
		return
	}
	selection.Dragging = false
	selection.DragOffsetCellX = 0
	selection.DragOffsetCellY = 0
	selection.DragSnapshotDone = false
}

func (s *EditorEntitySystem) hoveredEntityIndex(w *ecs.World, session *editorcomponent.EditorSession, pointer *editorcomponent.PointerState, items []levels.Entity, catalog *editorcomponent.PrefabCatalog) int {
	if pointer == nil || !pointer.InCanvas {
		return -1
	}
	indices := make([]int, 0, len(items))
	for index := range items {
		if !entitySelectableOnCurrentLayer(w, session, items[index]) {
			continue
		}
		indices = append(indices, index)
	}
	sort.SliceStable(indices, func(i, j int) bool {
		return compareEditorEntities(catalog, items, indices[i], indices[j]) > 0
	})
	for _, index := range indices {
		if s.entityContainsPoint(items[index], prefabInfoForEntity(catalog, items[index]), pointer.WorldX, pointer.WorldY) {
			return index
		}
	}
	return -1
}

func (s *EditorEntitySystem) entityContainsPoint(item levels.Entity, prefab *editorio.PrefabInfo, worldX, worldY float64) bool {
	left, top, width, height := entityBounds(item, prefab)
	if width <= 0 || height <= 0 {
		return false
	}
	return worldX >= left && worldX <= left+width && worldY >= top && worldY <= top+height
}

func (s *EditorEntitySystem) entityDraggable(item levels.Entity) bool {
	if strings.EqualFold(item.Type, "transition") || strings.EqualFold(item.Type, "gate") {
		return false
	}
	if item.Props == nil {
		return true
	}
	_, hasWidth := item.Props["w"]
	_, hasHeight := item.Props["h"]
	return !(hasWidth || hasHeight)
}

func entityBounds(item levels.Entity, prefab *editorio.PrefabInfo) (float64, float64, float64, float64) {
	prefab = resolvedPrefabInfoForItem(item, prefab)
	if item.Props != nil {
		width := toFloat(item.Props["w"])
		height := toFloat(item.Props["h"])
		if width > 0 || height > 0 {
			if width <= 0 {
				width = TileSize
			}
			if height <= 0 {
				height = TileSize
			}
			return float64(item.X), float64(item.Y), width, height
		}
	}
	width, height := prefabPreviewSize(prefab)
	originX, originY := prefabPreviewOrigin(prefab, width, height)
	anchorX, anchorY := entityAnchorPosition(item, originX, originY)
	scaleX, scaleY := entityPreviewScale(item, prefab)
	left := anchorX - originX*scaleX
	top := anchorY - originY*scaleY
	right := anchorX + (width-originX)*scaleX
	bottom := anchorY + (height-originY)*scaleY
	if left > right {
		left, right = right, left
	}
	if top > bottom {
		top, bottom = bottom, top
	}
	return left, top, right - left, bottom - top
}

func prefabPreviewSize(prefab *editorio.PrefabInfo) (float64, float64) {
	width := float64(TileSize)
	height := float64(TileSize)
	if prefab == nil {
		return width, height
	}
	if prefab.Preview.FrameW > 0 {
		width = float64(prefab.Preview.FrameW)
	}
	if prefab.Preview.FrameH > 0 {
		height = float64(prefab.Preview.FrameH)
	}
	if prefab.Preview.FallbackSize > 0 {
		if width <= 0 {
			width = float64(prefab.Preview.FallbackSize)
		}
		if height <= 0 {
			height = float64(prefab.Preview.FallbackSize)
		}
	}
	return width, height
}

func prefabPreviewOrigin(prefab *editorio.PrefabInfo, width, height float64) (float64, float64) {
	if prefab == nil {
		return 0, 0
	}
	originX := prefab.Preview.OriginX
	originY := prefab.Preview.OriginY
	if prefab.Preview.CenterOrigin && originX == 0 && originY == 0 {
		originX = width / 2
		originY = height / 2
	}
	return originX, originY
}

func prefabPreviewScale(prefab *editorio.PrefabInfo) (float64, float64) {
	scaleX := 1.0
	scaleY := 1.0
	if prefab == nil {
		return scaleX, scaleY
	}
	if prefab.Preview.ScaleX != 0 {
		scaleX = prefab.Preview.ScaleX
	}
	if prefab.Preview.ScaleY != 0 {
		scaleY = prefab.Preview.ScaleY
	}
	return scaleX, scaleY
}

func entityPreviewScale(item levels.Entity, prefab *editorio.PrefabInfo) (float64, float64) {
	scaleX, scaleY := prefabPreviewScale(prefab)
	if item.Props == nil {
		return scaleX, scaleY
	}
	if transformProps := entityComponentOverrideValues(item.Props, "transform"); transformProps != nil {
		if propsScaleX, ok := entityScaleValue(transformProps, "scale_x"); ok {
			scaleX = propsScaleX
		}
		if propsScaleY, ok := entityScaleValue(transformProps, "scale_y"); ok {
			scaleY = propsScaleY
		}
	}
	if propsScaleX, ok := entityScaleValue(item.Props, "scale_x"); ok {
		scaleX = propsScaleX
	}
	if propsScaleY, ok := entityScaleValue(item.Props, "scale_y"); ok {
		scaleY = propsScaleY
	}
	if rawTransform, ok := item.Props["transform"]; ok {
		if transformProps, ok := rawTransform.(map[string]interface{}); ok {
			if propsScaleX, ok := entityScaleValue(transformProps, "scale_x"); ok {
				scaleX = propsScaleX
			}
			if propsScaleY, ok := entityScaleValue(transformProps, "scale_y"); ok {
				scaleY = propsScaleY
			}
		}
	}
	return scaleX, scaleY
}

func entityScaleValue(props map[string]interface{}, key string) (float64, bool) {
	if props == nil {
		return 0, false
	}
	if _, ok := props[key]; !ok {
		return 0, false
	}
	return toFloat(props[key]), true
}

func entityAnchorPosition(item levels.Entity, originX, originY float64) (float64, float64) {
	anchorX := float64(item.X)
	anchorY := float64(item.Y)
	if isSpikeEntity(item) {
		anchorX += originX
		anchorY += originY
	}
	return anchorX, anchorY
}

func resolvedPrefabInfoForItem(item levels.Entity, prefab *editorio.PrefabInfo) *editorio.PrefabInfo {
	if prefab == nil {
		return nil
	}
	resolved := *prefab
	resolved.Preview = editorio.ResolvePrefabPreview(*prefab, entityComponentOverrides(item.Props))
	return &resolved
}

func toFloat(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func entityLabel(item levels.Entity) string {
	label := item.Type
	if strings.TrimSpace(item.ID) != "" {
		label = item.ID + " · " + label
	}
	layerLabel := ""
	if layer, ok := entityLayerIndex(item.Props); ok {
		layerLabel = fmt.Sprintf(" [L%d]", layer+1)
	}
	return fmt.Sprintf("%s @ (%d,%d)%s", label, item.X/TileSize, item.Y/TileSize, layerLabel)
}

var _ ecs.System = (*EditorEntitySystem)(nil)
