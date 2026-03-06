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
	_, selection, ok := entitySelectionState(w)
	if !ok || selection == nil {
		return
	}
	_, actions, _ := actionState(w)

	s.clampSelection(entities, selection)
	selection.HoveredIndex = s.hoveredEntityIndex(pointer, entities.Items, prefabs)

	if actions != nil {
		if strings.TrimSpace(actions.SelectPrefab) != "" {
			if info := prefabInfoByPath(prefabs, actions.SelectPrefab, ""); info != nil {
				placement.SelectedPath = info.Path
				placement.SelectedType = info.EntityType
				selection.SelectedIndex = -1
				selection.Dragging = false
				selection.DragSnapshotDone = false
				session.Status = "Selected prefab " + info.Name
			}
			actions.SelectPrefab = ""
		}
		if actions.SelectEntity >= 0 {
			if actions.SelectEntity < len(entities.Items) {
				selection.SelectedIndex = actions.SelectEntity
				selection.Dragging = false
				selection.DragSnapshotDone = false
				placement.SelectedPath = ""
				placement.SelectedType = ""
				session.Status = "Selected entity"
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
			selection.Dragging = false
			selection.DragSnapshotDone = false
			session.Status = "Cleared selection"
		}
	}

	if placement.SelectedPath != "" {
		if input.LeftJustPressed && pointer.HasCell {
			if s.placePrefab(w, session, entities, placement, pointer) {
				selection.SelectedIndex = len(entities.Items) - 1
				selection.Dragging = false
				selection.DragSnapshotDone = false
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
			selection.Dragging = s.entityDraggable(entities.Items[selection.SelectedIndex])
			selection.DragSnapshotDone = false
			session.Status = "Selected entity"
			return
		}
	}
}

func (s *EditorEntitySystem) clampSelection(entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState) {
	if entities == nil || selection == nil {
		return
	}
	if len(entities.Items) == 0 {
		selection.SelectedIndex = -1
		selection.HoveredIndex = -1
		selection.Dragging = false
		selection.DragSnapshotDone = false
		return
	}
	if selection.SelectedIndex >= len(entities.Items) {
		selection.SelectedIndex = len(entities.Items) - 1
	}
	if selection.SelectedIndex < -1 {
		selection.SelectedIndex = -1
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
	selection.Dragging = false
	selection.DragSnapshotDone = false
	setDirty(w, true)
	return true
}

func (s *EditorEntitySystem) updateEntityDrag(w *ecs.World, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, entities *editorcomponent.LevelEntities, selection *editorcomponent.EntitySelectionState) {
	if entities == nil || selection == nil || selection.SelectedIndex < 0 || selection.SelectedIndex >= len(entities.Items) {
		selection.Dragging = false
		selection.DragSnapshotDone = false
		return
	}
	if input.LeftJustReleased || !input.LeftDown || !pointer.InCanvas || !pointer.HasCell {
		selection.Dragging = false
		selection.DragSnapshotDone = false
		return
	}
	if !selection.DragSnapshotDone {
		pushSnapshot(w, "entity-drag")
		selection.DragSnapshotDone = true
	}
	item := &entities.Items[selection.SelectedIndex]
	nextX := pointer.CellX * TileSize
	nextY := pointer.CellY * TileSize
	if item.X != nextX || item.Y != nextY {
		item.X = nextX
		item.Y = nextY
		setDirty(w, true)
	}
}

func (s *EditorEntitySystem) hoveredEntityIndex(pointer *editorcomponent.PointerState, items []levels.Entity, catalog *editorcomponent.PrefabCatalog) int {
	if pointer == nil || !pointer.InCanvas {
		return -1
	}
	indices := make([]int, 0, len(items))
	for index := range items {
		indices = append(indices, index)
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left := prefabInfoForEntity(catalog, items[indices[i]])
		right := prefabInfoForEntity(catalog, items[indices[j]])
		leftLayer := 0
		rightLayer := 0
		if left != nil {
			leftLayer = left.Preview.RenderLayer
		}
		if right != nil {
			rightLayer = right.Preview.RenderLayer
		}
		if leftLayer == rightLayer {
			return indices[i] > indices[j]
		}
		return leftLayer > rightLayer
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
	width := float64(TileSize)
	height := float64(TileSize)
	originX := 0.0
	originY := 0.0
	if prefab != nil {
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
		originX = prefab.Preview.OriginX
		originY = prefab.Preview.OriginY
	}
	return float64(item.X) - originX, float64(item.Y) - originY, width, height
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
