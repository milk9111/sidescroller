package editorsystem

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/milk9111/sidescroller/cmd/editor/autotile"
	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

const (
	LeftPanelWidth      = 280.0
	RightPanelWidth     = 280.0
	TopToolbarHeight    = 56.0
	CanvasPadding       = 12.0
	TileSize            = model.DefaultTileSize
	entityComponentsKey = "components"
)

func sessionState(w *ecs.World) (ecs.Entity, *editorcomponent.EditorSession, bool) {
	entity, ok := ecs.First(w, editorcomponent.EditorSessionComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	session, ok := ecs.Get(w, entity, editorcomponent.EditorSessionComponent.Kind())
	return entity, session, ok && session != nil
}

func focusState(w *ecs.World) (ecs.Entity, *editorcomponent.EditorFocus, bool) {
	entity, ok := ecs.First(w, editorcomponent.EditorFocusComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	focus, ok := ecs.Get(w, entity, editorcomponent.EditorFocusComponent.Kind())
	return entity, focus, ok && focus != nil
}

func levelMetaState(w *ecs.World) (ecs.Entity, *editorcomponent.LevelMeta, bool) {
	entity, ok := ecs.First(w, editorcomponent.LevelMetaComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	meta, ok := ecs.Get(w, entity, editorcomponent.LevelMetaComponent.Kind())
	return entity, meta, ok && meta != nil
}

func cameraState(w *ecs.World) (ecs.Entity, *editorcomponent.CanvasCamera, bool) {
	entity, ok := ecs.First(w, editorcomponent.CanvasCameraComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	camera, ok := ecs.Get(w, entity, editorcomponent.CanvasCameraComponent.Kind())
	return entity, camera, ok && camera != nil
}

func rawInputState(w *ecs.World) (ecs.Entity, *editorcomponent.RawInputState, bool) {
	entity, ok := ecs.First(w, editorcomponent.RawInputStateComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	input, ok := ecs.Get(w, entity, editorcomponent.RawInputStateComponent.Kind())
	return entity, input, ok && input != nil
}

func pointerState(w *ecs.World) (ecs.Entity, *editorcomponent.PointerState, bool) {
	entity, ok := ecs.First(w, editorcomponent.PointerStateComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	pointer, ok := ecs.Get(w, entity, editorcomponent.PointerStateComponent.Kind())
	return entity, pointer, ok && pointer != nil
}

func strokeState(w *ecs.World) (ecs.Entity, *editorcomponent.ToolStroke, bool) {
	entity, ok := ecs.First(w, editorcomponent.ToolStrokeComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	stroke, ok := ecs.Get(w, entity, editorcomponent.ToolStrokeComponent.Kind())
	return entity, stroke, ok && stroke != nil
}

func undoState(w *ecs.World) (ecs.Entity, *editorcomponent.UndoStack, bool) {
	entity, ok := ecs.First(w, editorcomponent.UndoStackComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	undo, ok := ecs.Get(w, entity, editorcomponent.UndoStackComponent.Kind())
	return entity, undo, ok && undo != nil
}

func areaDragState(w *ecs.World) (ecs.Entity, *editorcomponent.AreaDragState, bool) {
	entity, ok := ecs.First(w, editorcomponent.AreaDragStateComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	state, ok := ecs.Get(w, entity, editorcomponent.AreaDragStateComponent.Kind())
	return entity, state, ok && state != nil
}

func overviewState(w *ecs.World) (ecs.Entity, *editorcomponent.OverviewState, bool) {
	entity, ok := ecs.First(w, editorcomponent.OverviewStateComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	state, ok := ecs.Get(w, entity, editorcomponent.OverviewStateComponent.Kind())
	return entity, state, ok && state != nil
}

func actionState(w *ecs.World) (ecs.Entity, *editorcomponent.EditorActions, bool) {
	entity, ok := ecs.First(w, editorcomponent.EditorActionsComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	actions, ok := ecs.Get(w, entity, editorcomponent.EditorActionsComponent.Kind())
	return entity, actions, ok && actions != nil
}

func reorderLayerEntities(w *ecs.World, currentIndex, nextIndex int) map[int]int {
	layers := layerEntities(w)
	if currentIndex < 0 || currentIndex >= len(layers) || nextIndex < 0 || nextIndex >= len(layers) || currentIndex == nextIndex {
		return nil
	}
	original := append([]ecs.Entity(nil), layers...)
	moving := layers[currentIndex]
	layers = append(layers[:currentIndex], layers[currentIndex+1:]...)
	if nextIndex >= len(layers) {
		layers = append(layers, moving)
	} else {
		layers = append(layers[:nextIndex], append([]ecs.Entity{moving}, layers[nextIndex:]...)...)
	}
	normalizeLayerOrders(w, layers)
	mapping := make(map[int]int, len(original))
	for newIndex, entity := range layers {
		for oldIndex, originalEntity := range original {
			if entity == originalEntity {
				mapping[oldIndex] = newIndex
				break
			}
		}
	}
	return mapping
}

func autotileState(w *ecs.World) (ecs.Entity, *editorcomponent.AutotileState, bool) {
	entity, ok := ecs.First(w, editorcomponent.AutotileStateComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	state, ok := ecs.Get(w, entity, editorcomponent.AutotileStateComponent.Kind())
	return entity, state, ok && state != nil
}

func catalogState(w *ecs.World) (ecs.Entity, *editorcomponent.TilesetCatalog, bool) {
	entity, ok := ecs.First(w, editorcomponent.TilesetCatalogComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	catalog, ok := ecs.Get(w, entity, editorcomponent.TilesetCatalogComponent.Kind())
	return entity, catalog, ok && catalog != nil
}

func prefabCatalogState(w *ecs.World) (ecs.Entity, *editorcomponent.PrefabCatalog, bool) {
	entity, ok := ecs.First(w, editorcomponent.PrefabCatalogComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	catalog, ok := ecs.Get(w, entity, editorcomponent.PrefabCatalogComponent.Kind())
	return entity, catalog, ok && catalog != nil
}

func prefabPlacementState(w *ecs.World) (ecs.Entity, *editorcomponent.PrefabPlacementState, bool) {
	entity, ok := ecs.First(w, editorcomponent.PrefabPlacementComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	state, ok := ecs.Get(w, entity, editorcomponent.PrefabPlacementComponent.Kind())
	return entity, state, ok && state != nil
}

func entitySelectionState(w *ecs.World) (ecs.Entity, *editorcomponent.EntitySelectionState, bool) {
	entity, ok := ecs.First(w, editorcomponent.EntitySelectionComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	state, ok := ecs.Get(w, entity, editorcomponent.EntitySelectionComponent.Kind())
	return entity, state, ok && state != nil
}

func entitiesState(w *ecs.World) (ecs.Entity, *editorcomponent.LevelEntities, bool) {
	entity, ok := ecs.First(w, editorcomponent.LevelEntitiesComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	items, ok := ecs.Get(w, entity, editorcomponent.LevelEntitiesComponent.Kind())
	return entity, items, ok && items != nil
}

func layerEntities(w *ecs.World) []ecs.Entity {
	entities := make([]ecs.Entity, 0)
	ecs.ForEach(w, editorcomponent.LayerDataComponent.Kind(), func(entity ecs.Entity, _ *editorcomponent.LayerData) {
		entities = append(entities, entity)
	})
	sort.Slice(entities, func(i, j int) bool {
		left, _ := ecs.Get(w, entities[i], editorcomponent.LayerDataComponent.Kind())
		right, _ := ecs.Get(w, entities[j], editorcomponent.LayerDataComponent.Kind())
		if left == nil || right == nil {
			return uint64(entities[i]) < uint64(entities[j])
		}
		return left.Order < right.Order
	})
	return entities
}

func layerAt(w *ecs.World, index int) (ecs.Entity, *editorcomponent.LayerData, bool) {
	layers := layerEntities(w)
	if index < 0 || index >= len(layers) {
		return 0, nil, false
	}
	layer, ok := ecs.Get(w, layers[index], editorcomponent.LayerDataComponent.Kind())
	return layers[index], layer, ok && layer != nil
}

func layerVisible(layer *editorcomponent.LayerData) bool {
	if layer == nil {
		return false
	}
	return !layer.Hidden
}

func layerIndexVisible(w *ecs.World, index int) bool {
	_, layer, ok := layerAt(w, index)
	if !ok {
		return true
	}
	return layerVisible(layer)
}

func entityVisibleOnLayer(w *ecs.World, item levels.Entity) bool {
	layerIndex, ok := entityLayerIndex(item.Props)
	if !ok {
		return true
	}
	return layerIndexVisible(w, layerIndex)
}

func entitySelectableOnCurrentLayer(w *ecs.World, session *editorcomponent.EditorSession, item levels.Entity) bool {
	if !entityVisibleOnLayer(w, item) {
		return false
	}
	if session == nil {
		return true
	}
	return normalizedEntityLayerIndex(item) == session.CurrentLayer
}

func normalizedEntityLayerIndex(item levels.Entity) int {
	layerIndex, ok := entityLayerIndex(item.Props)
	if !ok || layerIndex < 0 {
		return 0
	}
	return layerIndex
}

func editorEntityRenderOrder(catalog *editorcomponent.PrefabCatalog, item levels.Entity) int {
	prefab := prefabInfoForEntity(catalog, item)
	if prefab == nil {
		return 0
	}
	return prefab.Preview.RenderLayer
}

func compareEditorEntities(catalog *editorcomponent.PrefabCatalog, items []levels.Entity, leftIndex, rightIndex int) int {
	left := items[leftIndex]
	right := items[rightIndex]
	leftLayer := normalizedEntityLayerIndex(left)
	rightLayer := normalizedEntityLayerIndex(right)
	if leftLayer != rightLayer {
		if leftLayer < rightLayer {
			return -1
		}
		return 1
	}
	leftOrder := editorEntityRenderOrder(catalog, left)
	rightOrder := editorEntityRenderOrder(catalog, right)
	if leftOrder != rightOrder {
		if leftOrder < rightOrder {
			return -1
		}
		return 1
	}
	if leftIndex < rightIndex {
		return -1
	}
	if leftIndex > rightIndex {
		return 1
	}
	return 0
}

func setDirty(w *ecs.World, dirty bool) {
	if _, session, ok := sessionState(w); ok {
		session.Dirty = dirty
	}
	if _, meta, ok := levelMetaState(w); ok {
		meta.Dirty = dirty
	}
}

func layoutPanels(camera *editorcomponent.CanvasCamera) {
	if camera == nil {
		return
	}
	leftInset := effectiveLeftInset(camera)
	rightInset := effectiveRightInset(camera)
	topInset := effectiveTopInset(camera)
	camera.CanvasX = leftInset + CanvasPadding
	camera.CanvasY = topInset + CanvasPadding
	camera.CanvasW = math.Max(0, camera.ScreenW-leftInset-rightInset-(CanvasPadding*2))
	camera.CanvasH = math.Max(0, camera.ScreenH-topInset-(CanvasPadding*2))
}

func effectiveLeftInset(camera *editorcomponent.CanvasCamera) float64 {
	if camera != nil && camera.LeftInset > 0 {
		return camera.LeftInset
	}
	return LeftPanelWidth
}

func effectiveRightInset(camera *editorcomponent.CanvasCamera) float64 {
	if camera != nil && camera.RightInset > 0 {
		return camera.RightInset
	}
	return RightPanelWidth
}

func effectiveTopInset(camera *editorcomponent.CanvasCamera) float64 {
	if camera != nil && camera.TopInset > 0 {
		return camera.TopInset
	}
	return TopToolbarHeight
}

func refreshPointerFromCamera(pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, camera *editorcomponent.CanvasCamera, meta *editorcomponent.LevelMeta) {
	if pointer == nil || input == nil || camera == nil {
		return
	}
	leftInset := effectiveLeftInset(camera)
	rightInset := effectiveRightInset(camera)
	topInset := effectiveTopInset(camera)
	mouseX := float64(input.MouseX)
	mouseY := float64(input.MouseY)
	pointer.OverLeftPanel = mouseX < leftInset
	pointer.OverRightPanel = mouseX >= camera.ScreenW-rightInset
	pointer.OverToolbar = mouseY < topInset
	pointer.InCanvas = !pointer.OverLeftPanel && !pointer.OverRightPanel && !pointer.OverToolbar &&
		mouseX >= camera.CanvasX && mouseY >= camera.CanvasY &&
		mouseX < camera.CanvasX+camera.CanvasW && mouseY < camera.CanvasY+camera.CanvasH

	pointer.WorldX = camera.X + (mouseX-camera.CanvasX)/camera.Zoom
	pointer.WorldY = camera.Y + (mouseY-camera.CanvasY)/camera.Zoom
	pointer.CellX = int(math.Floor(pointer.WorldX / TileSize))
	pointer.CellY = int(math.Floor(pointer.WorldY / TileSize))
	pointer.HasCell = pointer.InCanvas && withinLevel(meta, pointer.CellX, pointer.CellY)
}

func withinLevel(meta *editorcomponent.LevelMeta, cellX, cellY int) bool {
	if meta == nil {
		return false
	}
	return cellX >= 0 && cellY >= 0 && cellX < meta.Width && cellY < meta.Height
}

func cellIndex(meta *editorcomponent.LevelMeta, cellX, cellY int) int {
	return cellY*meta.Width + cellX
}

func cloneCurrentLevel(w *ecs.World) model.LevelDocument {
	_, meta, _ := levelMetaState(w)
	doc := model.LevelDocument{}
	if meta != nil {
		doc.Width = meta.Width
		doc.Height = meta.Height
	}
	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer == nil {
			continue
		}
		doc.Layers = append(doc.Layers, model.Layer{
			Name:         layer.Name,
			Physics:      layer.Physics,
			Tiles:        append([]int(nil), layer.Tiles...),
			TilesetUsage: cloneUsage(layer.TilesetUsage),
		})
	}
	if _, entities, ok := entitiesState(w); ok && entities != nil {
		doc.Entities = append([]levels.Entity(nil), entities.Items...)
	}
	return doc
}

func restoreSnapshot(w *ecs.World, snapshot model.Snapshot) {
	for _, entity := range layerEntities(w) {
		ecs.DestroyEntity(w, entity)
	}
	for index, layer := range snapshot.Level.Layers {
		entity := ecs.CreateEntity(w)
		_ = ecs.Add(w, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
			Name:         layer.Name,
			Order:        index,
			Physics:      layer.Physics,
			Hidden:       false,
			Tiles:        append([]int(nil), layer.Tiles...),
			TilesetUsage: cloneUsage(layer.TilesetUsage),
		})
	}
	if _, meta, ok := levelMetaState(w); ok {
		meta.Width = snapshot.Level.Width
		meta.Height = snapshot.Level.Height
		meta.LoadedLevel = snapshot.LoadedLevel
		meta.Dirty = false
	}
	if _, entities, ok := entitiesState(w); ok {
		entities.Items = snapshot.Level.Clone().Entities
		normalizeEntityLayers(entities.Items)
	}
	if _, session, ok := sessionState(w); ok {
		session.CurrentLayer = clampInt(snapshot.CurrentLayer, 0, maxInt(0, len(snapshot.Level.Layers)-1))
		session.SaveTarget = snapshot.SaveTarget
		session.LoadedLevel = snapshot.LoadedLevel
		session.SelectedTile = snapshot.SelectedTile.Normalize()
		session.TransitionMode = false
		session.GateMode = false
		session.Status = "Undo applied"
		session.Dirty = false
	}
	resetTransientEditorState(w)
}

func pushSnapshot(w *ecs.World, reason string) {
	_, undo, ok := undoState(w)
	if !ok || undo == nil {
		return
	}
	_, session, hasSession := sessionState(w)
	if !hasSession || session == nil {
		return
	}
	current := cloneCurrentLevel(w)
	snapshot := model.Snapshot{
		Level:         current,
		CurrentLayer:  session.CurrentLayer,
		SaveTarget:    session.SaveTarget,
		LoadedLevel:   session.LoadedLevel,
		SelectedTile:  session.SelectedTile.Normalize(),
		StatusMessage: reason,
	}
	undo.Snapshots = append(undo.Snapshots, snapshot)
	if undo.Max <= 0 {
		undo.Max = 100
	}
	if len(undo.Snapshots) > undo.Max {
		undo.Snapshots = append([]model.Snapshot(nil), undo.Snapshots[len(undo.Snapshots)-undo.Max:]...)
	}
}

func resetTransientEditorState(w *ecs.World) {
	if _, stroke, ok := strokeState(w); ok && stroke != nil {
		stroke.Active = false
		stroke.Touched = nil
		stroke.Preview = nil
	}
	if _, drag, ok := areaDragState(w); ok && drag != nil {
		*drag = editorcomponent.AreaDragState{EntityIndex: -1}
	}
	if _, selection, ok := entitySelectionState(w); ok && selection != nil {
		selection.HoveredIndex = -1
		selection.Dragging = false
		selection.DragOffsetCellX = 0
		selection.DragOffsetCellY = 0
		selection.DragSnapshotDone = false
		selection.PropertySnapshotDone = false
	}
	if _, placement, ok := prefabPlacementState(w); ok && placement != nil {
		placement.SelectedPath = ""
		placement.SelectedType = ""
	}
}

func cloneUsage(input []*levels.TileInfo) []*levels.TileInfo {
	output := make([]*levels.TileInfo, len(input))
	for index, item := range input {
		if item == nil {
			continue
		}
		copied := *item
		output[index] = &copied
	}
	return output
}

func autotileEnabled(w *ecs.World) bool {
	_, state, ok := autotileState(w)
	return ok && state != nil && state.Enabled
}

func queueAutotileCell(w *ecs.World, layerIndex, cellIndex int) {
	_, state, ok := autotileState(w)
	if !ok || state == nil {
		return
	}
	if state.DirtyCells == nil {
		state.DirtyCells = make(map[int]map[int]struct{})
	}
	if state.DirtyCells[layerIndex] == nil {
		state.DirtyCells[layerIndex] = make(map[int]struct{})
	}
	state.DirtyCells[layerIndex][cellIndex] = struct{}{}
}

func queueAutotileNeighborhood(w *ecs.World, layerIndex int, meta *editorcomponent.LevelMeta, cellX, cellY int) {
	if meta == nil {
		return
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nextX := cellX + dx
			nextY := cellY + dy
			if !withinLevel(meta, nextX, nextY) {
				continue
			}
			queueAutotileCell(w, layerIndex, cellIndex(meta, nextX, nextY))
		}
	}
}

func queueAutotileFullLayer(w *ecs.World, layerIndex int) {
	_, state, ok := autotileState(w)
	if !ok || state == nil {
		return
	}
	if state.FullRebuild == nil {
		state.FullRebuild = make(map[int]bool)
	}
	state.FullRebuild[layerIndex] = true
}

func clearAutotileQueues(state *editorcomponent.AutotileState) {
	if state == nil {
		return
	}
	state.DirtyCells = make(map[int]map[int]struct{})
	state.FullRebuild = make(map[int]bool)
}

func normalizeLayerOrders(w *ecs.World, ordered []ecs.Entity) {
	for index, entity := range ordered {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer != nil {
			layer.Order = index
		}
	}
}

func nextLayerName(w *ecs.World) string {
	existing := make(map[string]struct{})
	for _, entity := range layerEntities(w) {
		if layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind()); layer != nil {
			existing[strings.TrimSpace(layer.Name)] = struct{}{}
		}
	}
	for index := 1; ; index++ {
		candidate := fmt.Sprintf("Layer %d", index)
		if _, exists := existing[candidate]; !exists {
			return candidate
		}
	}
}

func remapEntityLayerProps(items []levels.Entity, mapping map[int]int) {
	for index := range items {
		layerIndex, ok := entityLayerIndex(items[index].Props)
		if !ok {
			continue
		}
		mapped, exists := mapping[layerIndex]
		if !exists {
			continue
		}
		if items[index].Props == nil {
			items[index].Props = make(map[string]interface{})
		}
		items[index].Props["layer"] = mapped
	}
}

func normalizeEntityLayers(items []levels.Entity) {
	for index := range items {
		layerIndex, ok := entityLayerIndex(items[index].Props)
		if !ok || layerIndex < 0 {
			layerIndex = 0
		}
		if items[index].Props == nil {
			items[index].Props = make(map[string]interface{})
		}
		items[index].Props["layer"] = layerIndex
	}
}

func entityLayerIndex(props map[string]interface{}) (int, bool) {
	if props == nil {
		return 0, false
	}
	raw, ok := props["layer"]
	if !ok {
		return 0, false
	}
	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float32:
		return int(value), true
	case float64:
		return int(value), true
	default:
		return 0, false
	}
}

func prefabPathForEntity(item levels.Entity) string {
	if item.Props != nil {
		if prefabPath, ok := item.Props["prefab"].(string); ok && strings.TrimSpace(prefabPath) != "" {
			return strings.TrimSpace(prefabPath)
		}
	}
	if strings.TrimSpace(item.Type) == "" {
		return ""
	}
	return strings.TrimSpace(item.Type) + ".yaml"
}

func prefabInfoForEntity(catalog *editorcomponent.PrefabCatalog, item levels.Entity) *editorio.PrefabInfo {
	if catalog == nil {
		return nil
	}
	return prefabInfoByPath(catalog, prefabPathForEntity(item), item.Type)
}

func prefabInfoByPath(catalog *editorcomponent.PrefabCatalog, path, entityType string) *editorio.PrefabInfo {
	if catalog == nil {
		return nil
	}
	cleanPath := strings.TrimSpace(path)
	cleanType := strings.TrimSpace(entityType)
	for index := range catalog.Items {
		item := &catalog.Items[index]
		if cleanPath != "" && item.Path == cleanPath {
			return item
		}
	}
	for index := range catalog.Items {
		item := &catalog.Items[index]
		if cleanType != "" && item.EntityType == cleanType {
			return item
		}
	}
	return nil
}

func autotileGroupEqual(left, right *levels.TileInfo) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Auto == right.Auto && left.Path == right.Path && left.BaseIndex == right.BaseIndex && left.TileW == right.TileW && left.TileH == right.TileH
}

func autotileMaskFor(layer *editorcomponent.LayerData, meta *editorcomponent.LevelMeta, cellX, cellY int, usage *levels.TileInfo) uint8 {
	connected := func(x, y int) bool {
		if !withinLevel(meta, x, y) {
			return false
		}
		neighbor := layer.TilesetUsage[cellIndex(meta, x, y)]
		return neighbor != nil && neighbor.Auto && autotileGroupEqual(neighbor, usage)
	}
	north := connected(cellX, cellY-1)
	east := connected(cellX+1, cellY)
	south := connected(cellX, cellY+1)
	west := connected(cellX-1, cellY)
	return autotile.BuildMask(
		north,
		east,
		south,
		west,
		north && west && connected(cellX-1, cellY-1),
		north && east && connected(cellX+1, cellY-1),
		south && east && connected(cellX+1, cellY+1),
		south && west && connected(cellX-1, cellY+1),
	)
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isTransitionEntity(item levels.Entity) bool {
	return strings.EqualFold(strings.TrimSpace(item.Type), "transition")
}

func isGateEntity(item levels.Entity) bool {
	return strings.EqualFold(strings.TrimSpace(item.Type), "gate")
}

func isSpikeEntity(item levels.Entity) bool {
	return strings.EqualFold(strings.TrimSpace(item.Type), "spike")
}

func selectedSpecialEntity(items []levels.Entity, selection *editorcomponent.EntitySelectionState, kind string) (int, *levels.Entity) {
	if selection == nil || selection.SelectedIndex < 0 || selection.SelectedIndex >= len(items) {
		return -1, nil
	}
	item := &items[selection.SelectedIndex]
	switch kind {
	case "transition":
		if isTransitionEntity(*item) {
			return selection.SelectedIndex, item
		}
	case "gate":
		if isGateEntity(*item) {
			return selection.SelectedIndex, item
		}
	}
	return -1, nil
}

func ensureEntityProps(item *levels.Entity) map[string]interface{} {
	if item == nil {
		return nil
	}
	if item.Props == nil {
		item.Props = make(map[string]interface{})
	}
	return item.Props
}

func entityComponentOverrides(props map[string]interface{}) map[string]any {
	if props == nil {
		return nil
	}
	raw, ok := props[entityComponentsKey]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case map[string]interface{}:
		converted := make(map[string]any, len(typed))
		for key, value := range typed {
			converted[key] = value
		}
		return converted
	default:
		return nil
	}
}

func entityComponentOverrideValues(props map[string]interface{}, componentName string) map[string]any {
	overrides := entityComponentOverrides(props)
	if overrides == nil {
		return nil
	}
	raw, ok := overrides[componentName]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case map[string]interface{}:
		converted := make(map[string]any, len(typed))
		for key, value := range typed {
			converted[key] = value
		}
		return converted
	default:
		return nil
	}
}

func ensureEntityComponentOverrideValues(item *levels.Entity, componentName string) map[string]any {
	props := ensureEntityProps(item)
	raw, ok := props[entityComponentsKey]
	if !ok || raw == nil {
		overrides := make(map[string]any)
		props[entityComponentsKey] = overrides
		componentValues := make(map[string]any)
		overrides[componentName] = componentValues
		return componentValues
	}
	overrides, ok := raw.(map[string]any)
	if !ok {
		if converted, convertedOK := raw.(map[string]interface{}); convertedOK {
			overrides = make(map[string]any, len(converted))
			for key, value := range converted {
				overrides[key] = value
			}
			props[entityComponentsKey] = overrides
		} else {
			overrides = make(map[string]any)
			props[entityComponentsKey] = overrides
		}
	}
	if rawComponent, ok := overrides[componentName]; ok && rawComponent != nil {
		switch typed := rawComponent.(type) {
		case map[string]interface{}:
			converted := make(map[string]any, len(typed))
			for key, value := range typed {
				converted[key] = value
			}
			overrides[componentName] = converted
			return converted
		}
	}
	componentValues := make(map[string]any)
	overrides[componentName] = componentValues
	return componentValues
}

func ensureUniqueEntityIDs(items []levels.Entity) bool {
	used := make(map[string]struct{}, len(items))
	nextByPrefix := make(map[string]int)
	nextTransition := 1
	changed := false
	for index := range items {
		candidate := canonicalEntityID(items[index])
		if candidate != "" {
			if _, exists := used[candidate]; !exists {
				used[candidate] = struct{}{}
				if syncEntityID(&items[index], candidate) {
					changed = true
				}
				continue
			}
		}
		replacement := nextGeneratedEntityID(items[index], used, nextByPrefix, &nextTransition)
		used[replacement] = struct{}{}
		if syncEntityID(&items[index], replacement) {
			changed = true
		}
	}
	return changed
}

func canonicalEntityID(item levels.Entity) string {
	id := strings.TrimSpace(item.ID)
	if id != "" {
		return id
	}
	return entityStringProp(item, "id")
}

func syncEntityID(item *levels.Entity, id string) bool {
	if item == nil {
		return false
	}
	changed := strings.TrimSpace(item.ID) != id
	item.ID = id
	if isTransitionEntity(*item) || hasStringProp(item.Props, "id") {
		props := ensureEntityProps(item)
		if value, _ := props["id"].(string); value != id {
			props["id"] = id
			changed = true
		}
	}
	return changed
}

func hasStringProp(props map[string]interface{}, key string) bool {
	if props == nil {
		return false
	}
	_, ok := props[key].(string)
	return ok
}

func nextGeneratedEntityID(item levels.Entity, used map[string]struct{}, nextByPrefix map[string]int, nextTransition *int) string {
	if isTransitionEntity(item) {
		for {
			candidate := fmt.Sprintf("t%d", *nextTransition)
			*nextTransition++
			if _, exists := used[candidate]; !exists {
				return candidate
			}
		}
	}
	prefix := sanitizeEntityIDPrefix(item.Type)
	for {
		nextByPrefix[prefix]++
		candidate := fmt.Sprintf("%s_%d", prefix, nextByPrefix[prefix])
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

func sanitizeEntityIDPrefix(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "entity"
	}
	var builder strings.Builder
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			if builder.Len() == 0 || strings.HasSuffix(builder.String(), "_") {
				continue
			}
			builder.WriteByte('_')
		}
	}
	prefix := strings.Trim(builder.String(), "_")
	if prefix == "" {
		return "entity"
	}
	return prefix
}

func entityStringProp(item levels.Entity, key string) string {
	if item.Props == nil {
		return ""
	}
	if value, ok := item.Props[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func spikeRotationForCell(w *ecs.World, meta *editorcomponent.LevelMeta, cellX, cellY int) float64 {
	if meta == nil {
		return 0
	}
	supportChecks := []struct {
		nextX    int
		nextY    int
		boundary bool
		rotation float64
	}{
		{nextX: cellX, nextY: cellY + 1, boundary: cellY >= meta.Height-1, rotation: 0},
		{nextX: cellX - 1, nextY: cellY, boundary: cellX <= 0, rotation: 90},
		{nextX: cellX, nextY: cellY - 1, boundary: cellY <= 0, rotation: 180},
		{nextX: cellX + 1, nextY: cellY, boundary: cellX >= meta.Width-1, rotation: 270},
	}
	for _, check := range supportChecks {
		if check.boundary || solidCellAt(w, meta, check.nextX, check.nextY) {
			return check.rotation
		}
	}
	return 0
}

func solidCellAt(w *ecs.World, meta *editorcomponent.LevelMeta, cellX, cellY int) bool {
	if meta == nil || !withinLevel(meta, cellX, cellY) {
		return false
	}
	index := cellIndex(meta, cellX, cellY)
	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer == nil || !layer.Physics || index < 0 || index >= len(layer.Tiles) {
			continue
		}
		if layer.Tiles[index] != 0 {
			return true
		}
	}
	return false
}

func entityRect(item levels.Entity) (float64, float64, float64, float64) {
	left := float64(item.X)
	top := float64(item.Y)
	width := toFloat(item.Props["w"])
	height := toFloat(item.Props["h"])
	if width <= 0 {
		width = TileSize
	}
	if height <= 0 {
		height = TileSize
	}
	return left, top, width, height
}

func applyLoadedLevel(w *ecs.World, normalized string, doc *model.LevelDocument) {
	if doc == nil {
		return
	}
	for _, entity := range layerEntities(w) {
		ecs.DestroyEntity(w, entity)
	}
	for index, layer := range doc.Layers {
		entity := ecs.CreateEntity(w)
		_ = ecs.Add(w, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
			Name:         layer.Name,
			Order:        index,
			Physics:      layer.Physics,
			Hidden:       false,
			Tiles:        append([]int(nil), layer.Tiles...),
			TilesetUsage: cloneUsage(layer.TilesetUsage),
		})
	}
	if _, entities, ok := entitiesState(w); ok && entities != nil {
		entities.Items = doc.Clone().Entities
		normalizeEntityLayers(entities.Items)
	}
	if _, meta, ok := levelMetaState(w); ok && meta != nil {
		meta.Width = doc.Width
		meta.Height = doc.Height
		meta.LoadedLevel = normalized
		meta.Dirty = false
	}
	if _, session, ok := sessionState(w); ok && session != nil {
		session.SaveTarget = normalized
		session.LoadedLevel = normalized
		session.CurrentLayer = clampInt(session.CurrentLayer, 0, maxInt(0, len(doc.Layers)-1))
		session.Dirty = false
		session.TransitionMode = false
		session.GateMode = false
	}
	if _, overview, ok := overviewState(w); ok && overview != nil {
		overview.NeedsRefresh = true
		overview.LoadLevel = ""
	}
	clearAutotileOnLoad(w)
	resetTransientEditorState(w)
}

func clearAutotileOnLoad(w *ecs.World) {
	if _, state, ok := autotileState(w); ok && state != nil {
		clearAutotileQueues(state)
	}
}

func normalizeLevelRef(value string) string {
	return editorio.NormalizeLevelTarget(strings.TrimSpace(value))
}
