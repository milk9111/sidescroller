package editorsystem

import (
	"container/list"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

type EditorToolSystem struct{}

func NewEditorToolSystem() *EditorToolSystem {
	return &EditorToolSystem{}
}

func (s *EditorToolSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok {
		return
	}
	_, input, ok := rawInputState(w)
	if !ok {
		return
	}
	_, pointer, ok := pointerState(w)
	if !ok {
		return
	}
	_, meta, ok := levelMetaState(w)
	if !ok {
		return
	}
	_, stroke, ok := strokeState(w)
	if !ok {
		return
	}
	_, moveSelection, _ := moveSelectionState(w)
	_, placement, _ := prefabPlacementState(w)
	_, selection, _ := entitySelectionState(w)
	_, prefabCatalog, _ := prefabCatalogState(w)

	if session.OverviewOpen || session.TransitionMode || session.GateMode || session.TriggerMode || session.BreakableWallMode {
		stroke.Active = false
		stroke.Touched = nil
		stroke.Preview = nil
		if moveSelection != nil {
			moveSelection.Selecting = false
			moveSelection.Moving = false
		}
		return
	}

	clampCurrentLayer(w, session)
	updateIdlePreview(stroke, session, pointer)
	if placement != nil && placement.SelectedPath != "" {
		if input.LeftJustReleased {
			stroke.Active = false
			stroke.Preview = nil
		}
		return
	}
	if session.ActiveTool != editorcomponent.ToolMove && selection != nil && (selection.Dragging || (selection.HoveredIndex >= 0 && input.LeftJustPressed)) {
		if input.LeftJustReleased {
			stroke.Active = false
			stroke.Preview = nil
		}
		return
	}

	if input.RightJustPressed && pointer.HasCell {
		s.sampleTileAtCursor(w, session, meta, pointer)
	}

	if !pointer.InCanvas {
		if input.LeftJustReleased {
			stroke.Active = false
			stroke.Preview = nil
		}
		return
	}

	if input.LeftJustPressed {
		clearLayerDeleteArm(w)
	}

	switch session.ActiveTool {
	case editorcomponent.ToolBrush:
		s.updatePaintStroke(w, session, meta, pointer, input, stroke, false)
	case editorcomponent.ToolErase:
		s.updatePaintStroke(w, session, meta, pointer, input, stroke, true)
	case editorcomponent.ToolFill:
		if input.LeftJustPressed && pointer.HasCell {
			pushSnapshot(w, "fill")
			if s.fillLayer(w, session, meta, pointer.CellX, pointer.CellY) {
				setDirty(w, true)
				session.Status = "Filled region"
			}
		}
	case editorcomponent.ToolBox:
		s.updateBoxStroke(w, session, meta, pointer, input, stroke, false)
	case editorcomponent.ToolBoxErase:
		s.updateBoxStroke(w, session, meta, pointer, input, stroke, true)
	case editorcomponent.ToolLine:
		s.updateLineStroke(w, session, meta, pointer, input, stroke)
	case editorcomponent.ToolMove:
		s.updateMoveSelection(w, session, meta, pointer, input, stroke, moveSelection, prefabCatalog)
	case editorcomponent.ToolSpike:
		s.updateSpikeStroke(w, session, meta, pointer, input, stroke)
	}
}

func (s *EditorToolSystem) updateMoveSelection(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, stroke *editorcomponent.ToolStroke, state *editorcomponent.MoveSelectionState, catalog *editorcomponent.PrefabCatalog) {
	if state == nil {
		return
	}
	if !pointer.InCanvas {
		if input.LeftJustReleased {
			stroke.Active = false
			stroke.Preview = nil
			state.Selecting = false
			state.Moving = false
		}
		return
	}
	if state.Active && !state.Selecting && !state.Moving {
		stroke.Active = false
		stroke.Preview = nil
		if input.LeftJustPressed && pointer.HasCell {
			if moveSelectionContainsCell(state, pointer.CellX, pointer.CellY) {
				state.Moving = true
				state.DragOffsetCellX = pointer.CellX - state.DestMinX
				state.DragOffsetCellY = pointer.CellY - state.DestMinY
				s.updateMoveDestination(meta, state, state.DestMinX, state.DestMinY)
				session.Status = "Moving room"
				return
			}
			*state = editorcomponent.MoveSelectionState{}
		}
	}
	if state.Moving {
		if pointer.HasCell {
			nextMinX := pointer.CellX - state.DragOffsetCellX
			nextMinY := pointer.CellY - state.DragOffsetCellY
			s.updateMoveDestination(meta, state, nextMinX, nextMinY)
		}
		if input.LeftJustReleased || !input.LeftDown {
			if state.DestMinX != state.SourceMinX || state.DestMinY != state.SourceMinY {
				pushSnapshot(w, "move-room")
				if s.applyMoveSelection(w, session, meta, state) {
					setDirty(w, true)
					session.Status = "Moved room"
				}
			} else {
				session.Status = "Room move ready"
			}
			stroke.Active = false
			stroke.Preview = nil
			state.Moving = false
			state.Active = state.Active && (state.DestMinX == state.SourceMinX && state.DestMinY == state.SourceMinY)
			if !state.Active {
				*state = editorcomponent.MoveSelectionState{}
			}
		}
		return
	}
	if input.LeftJustPressed && pointer.HasCell {
		stroke.Active = true
		stroke.Tool = editorcomponent.ToolMove
		stroke.StartCellX = pointer.CellX
		stroke.StartCellY = pointer.CellY
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = filledRectCells(pointer.CellX, pointer.CellY, pointer.CellX, pointer.CellY)
		state.Active = false
		state.Selecting = true
		state.Moving = false
		state.StartCellX = pointer.CellX
		state.StartCellY = pointer.CellY
		return
	}
	if state.Selecting && stroke.Active && input.LeftDown && pointer.HasCell {
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = filledRectCells(state.StartCellX, state.StartCellY, pointer.CellX, pointer.CellY)
	}
	if state.Selecting && stroke.Active && input.LeftJustReleased {
		minX, minY, width, height := rectFromCells(state.StartCellX, state.StartCellY, stroke.LastCellX, stroke.LastCellY)
		if width > 0 && height > 0 && s.captureMoveSelection(w, meta, state, catalog, minX, minY, width, height) {
			state.Active = true
			state.Selecting = false
			state.Moving = false
			state.SourceMinX = minX
			state.SourceMinY = minY
			state.Width = width
			state.Height = height
			state.DestMinX = minX
			state.DestMinY = minY
			session.Status = "Room selected; drag to move"
		} else {
			*state = editorcomponent.MoveSelectionState{}
		}
		stroke.Active = false
		stroke.Preview = nil
	}
}

func (s *EditorToolSystem) updatePaintStroke(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, stroke *editorcomponent.ToolStroke, erase bool) {
	if input.LeftJustPressed && pointer.HasCell {
		pushSnapshot(w, string(session.ActiveTool))
		stroke.Active = true
		stroke.Tool = session.ActiveTool
		stroke.StartCellX = pointer.CellX
		stroke.StartCellY = pointer.CellY
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Touched = map[int]struct{}{}
		s.applyLineToLayer(w, session, meta, stroke, pointer.CellX, pointer.CellY, erase)
		setDirty(w, true)
	}
	if stroke.Active && input.LeftDown && pointer.HasCell {
		s.applyLineToLayer(w, session, meta, stroke, pointer.CellX, pointer.CellY, erase)
	}
	if stroke.Active && (input.LeftJustReleased || !input.LeftDown) {
		stroke.Active = false
		stroke.Touched = nil
	}
}

func (s *EditorToolSystem) updateLineStroke(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, stroke *editorcomponent.ToolStroke) {
	if input.LeftJustPressed && pointer.HasCell {
		pushSnapshot(w, "line")
		stroke.Active = true
		stroke.Tool = editorcomponent.ToolLine
		stroke.StartCellX = pointer.CellX
		stroke.StartCellY = pointer.CellY
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = bresenhamCells(pointer.CellX, pointer.CellY, pointer.CellX, pointer.CellY)
		return
	}
	if stroke.Active && input.LeftDown && pointer.HasCell {
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = bresenhamCells(stroke.StartCellX, stroke.StartCellY, pointer.CellX, pointer.CellY)
	}
	if stroke.Active && input.LeftJustReleased {
		changed := false
		for _, cell := range stroke.Preview {
			if !withinLevel(meta, cell.X, cell.Y) {
				continue
			}
			if s.applyTileAt(w, session, meta, cell.X, cell.Y, false) {
				changed = true
			}
		}
		if changed {
			setDirty(w, true)
			session.Status = "Line placed"
		}
		stroke.Active = false
		stroke.Preview = nil
	}
}

func (s *EditorToolSystem) updateBoxStroke(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, stroke *editorcomponent.ToolStroke, erase bool) {
	if input.LeftJustPressed && pointer.HasCell {
		action := "box"
		if erase {
			action = "box erase"
		}
		pushSnapshot(w, action)
		stroke.Active = true
		stroke.Tool = session.ActiveTool
		stroke.StartCellX = pointer.CellX
		stroke.StartCellY = pointer.CellY
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = filledRectCells(pointer.CellX, pointer.CellY, pointer.CellX, pointer.CellY)
		return
	}
	if stroke.Active && input.LeftDown && pointer.HasCell {
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = filledRectCells(stroke.StartCellX, stroke.StartCellY, pointer.CellX, pointer.CellY)
	}
	if stroke.Active && input.LeftJustReleased {
		changed := false
		for _, cell := range stroke.Preview {
			if !withinLevel(meta, cell.X, cell.Y) {
				continue
			}
			if s.applyTileAt(w, session, meta, cell.X, cell.Y, erase) {
				changed = true
			}
		}
		if changed {
			setDirty(w, true)
			if erase {
				session.Status = "Box erased"
			} else {
				session.Status = "Box placed"
			}
		}
		stroke.Active = false
		stroke.Preview = nil
	}
}

func (s *EditorToolSystem) updateSpikeStroke(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState, input *editorcomponent.RawInputState, stroke *editorcomponent.ToolStroke) {
	if input.LeftJustPressed && pointer.HasCell {
		pushSnapshot(w, "spike")
		stroke.Active = true
		stroke.Tool = editorcomponent.ToolSpike
		stroke.StartCellX = pointer.CellX
		stroke.StartCellY = pointer.CellY
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = bresenhamCells(pointer.CellX, pointer.CellY, pointer.CellX, pointer.CellY)
		return
	}
	if stroke.Active && input.LeftDown && pointer.HasCell {
		stroke.LastCellX = pointer.CellX
		stroke.LastCellY = pointer.CellY
		stroke.Preview = bresenhamCells(stroke.StartCellX, stroke.StartCellY, pointer.CellX, pointer.CellY)
	}
	if stroke.Active && input.LeftJustReleased {
		changed := false
		for _, cell := range stroke.Preview {
			if !withinLevel(meta, cell.X, cell.Y) {
				continue
			}
			if s.applySpikeAt(w, session, meta, cell.X, cell.Y) {
				changed = true
			}
		}
		if changed {
			setDirty(w, true)
			session.Status = "Placed spikes"
		}
		stroke.Active = false
		stroke.Preview = nil
	}
}

func (s *EditorToolSystem) applyLineToLayer(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, stroke *editorcomponent.ToolStroke, cellX, cellY int, erase bool) {
	for _, cell := range bresenhamCells(stroke.LastCellX, stroke.LastCellY, cellX, cellY) {
		if !withinLevel(meta, cell.X, cell.Y) {
			continue
		}
		index := cellIndex(meta, cell.X, cell.Y)
		if _, exists := stroke.Touched[index]; exists {
			continue
		}
		if s.applyTileAt(w, session, meta, cell.X, cell.Y, erase) {
			stroke.Touched[index] = struct{}{}
		}
	}
	stroke.LastCellX = cellX
	stroke.LastCellY = cellY
}

func (s *EditorToolSystem) fillLayer(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, startX, startY int) bool {
	if autotileEnabled(w) {
		return s.fillAutotileLayer(w, session, meta, startX, startY)
	}
	return s.fillRawLayer(w, session, meta, startX, startY)
}

func (s *EditorToolSystem) fillRawLayer(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, startX, startY int) bool {
	_, layer, ok := layerAt(w, session.CurrentLayer)
	if !ok || layer == nil || !withinLevel(meta, startX, startY) {
		return false
	}
	startIndex := cellIndex(meta, startX, startY)
	targetValue := layer.Tiles[startIndex]
	newValue, newUsage := selectedTileValue(w, session)
	if targetValue == newValue {
		return false
	}

	queue := list.New()
	queue.PushBack(editorcomponent.GridCell{X: startX, Y: startY})
	visited := map[int]struct{}{startIndex: {}}
	changed := false
	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(editorcomponent.GridCell)
		if !withinLevel(meta, current.X, current.Y) {
			continue
		}
		index := cellIndex(meta, current.X, current.Y)
		if layer.Tiles[index] != targetValue {
			continue
		}
		layer.Tiles[index] = newValue
		layer.TilesetUsage[index] = cloneTileUsage(newUsage)
		changed = true
		neighbors := []editorcomponent.GridCell{{X: current.X + 1, Y: current.Y}, {X: current.X - 1, Y: current.Y}, {X: current.X, Y: current.Y + 1}, {X: current.X, Y: current.Y - 1}}
		for _, next := range neighbors {
			if !withinLevel(meta, next.X, next.Y) {
				continue
			}
			nextIndex := cellIndex(meta, next.X, next.Y)
			if _, seen := visited[nextIndex]; seen {
				continue
			}
			visited[nextIndex] = struct{}{}
			queue.PushBack(next)
		}
	}
	return changed
}

func (s *EditorToolSystem) fillAutotileLayer(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, startX, startY int) bool {
	_, layer, ok := layerAt(w, session.CurrentLayer)
	if !ok || layer == nil || !withinLevel(meta, startX, startY) {
		return false
	}
	startIndex := cellIndex(meta, startX, startY)
	targetUsage := cloneTileUsage(layer.TilesetUsage[startIndex])
	_, newUsage := selectedTileValue(w, session)
	if newUsage == nil {
		return false
	}
	if autotileGroupEqual(targetUsage, newUsage) {
		return false
	}

	queue := list.New()
	queue.PushBack(editorcomponent.GridCell{X: startX, Y: startY})
	visited := map[int]struct{}{startIndex: {}}
	changed := false
	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(editorcomponent.GridCell)
		if !withinLevel(meta, current.X, current.Y) {
			continue
		}
		index := cellIndex(meta, current.X, current.Y)
		if !autotileGroupEqual(layer.TilesetUsage[index], targetUsage) {
			continue
		}
		if !tileUsageEqual(layer.TilesetUsage[index], newUsage) {
			layer.TilesetUsage[index] = cloneTileUsage(newUsage)
			layer.Tiles[index] = newUsage.Index
			changed = true
		}
		neighbors := []editorcomponent.GridCell{{X: current.X + 1, Y: current.Y}, {X: current.X - 1, Y: current.Y}, {X: current.X, Y: current.Y + 1}, {X: current.X, Y: current.Y - 1}}
		for _, next := range neighbors {
			if !withinLevel(meta, next.X, next.Y) {
				continue
			}
			nextIndex := cellIndex(meta, next.X, next.Y)
			if _, seen := visited[nextIndex]; seen {
				continue
			}
			visited[nextIndex] = struct{}{}
			queue.PushBack(next)
		}
	}
	if changed {
		queueAutotileFullLayer(w, session.CurrentLayer)
	}
	return changed
}

func (s *EditorToolSystem) applyTileAt(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, cellX, cellY int, erase bool) bool {
	_, layer, ok := layerAt(w, session.CurrentLayer)
	if !ok || layer == nil || !withinLevel(meta, cellX, cellY) {
		return false
	}
	index := cellIndex(meta, cellX, cellY)
	if erase {
		if layer.Tiles[index] == 0 && layer.TilesetUsage[index] == nil {
			return false
		}
		layer.Tiles[index] = 0
		layer.TilesetUsage[index] = nil
		if autotileEnabled(w) {
			queueAutotileNeighborhood(w, session.CurrentLayer, meta, cellX, cellY)
		}
		return true
	}
	newValue, newUsage := selectedTileValue(w, session)
	if session.SelectedTile.Path == "" {
		session.Status = "No tileset selected"
		return false
	}
	if layer.Tiles[index] == newValue && tileUsageEqual(layer.TilesetUsage[index], newUsage) {
		return false
	}
	layer.Tiles[index] = newValue
	layer.TilesetUsage[index] = cloneTileUsage(newUsage)
	if autotileEnabled(w) {
		queueAutotileNeighborhood(w, session.CurrentLayer, meta, cellX, cellY)
	}
	return true
}

func (s *EditorToolSystem) captureMoveSelection(w *ecs.World, meta *editorcomponent.LevelMeta, state *editorcomponent.MoveSelectionState, catalog *editorcomponent.PrefabCatalog, minX, minY, width, height int) bool {
	if meta == nil || state == nil || width <= 0 || height <= 0 {
		return false
	}
	state.Layers = state.Layers[:0]
	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer == nil {
			continue
		}
		selection := editorcomponent.MoveLayerSelection{
			Tiles:        make([]int, width*height),
			TilesetUsage: make([]*levels.TileInfo, width*height),
		}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				sourceIndex := cellIndex(meta, minX+x, minY+y)
				targetIndex := y*width + x
				selection.Tiles[targetIndex] = layer.Tiles[sourceIndex]
				selection.TilesetUsage[targetIndex] = cloneTileUsage(layer.TilesetUsage[sourceIndex])
			}
		}
		state.Layers = append(state.Layers, selection)
	}
	state.Entities = state.Entities[:0]
	if _, entities, ok := entitiesState(w); ok && entities != nil {
		left := float64(minX * TileSize)
		top := float64(minY * TileSize)
		right := float64((minX + width) * TileSize)
		bottom := float64((minY + height) * TileSize)
		for index, item := range entities.Items {
			entityLeft, entityTop, entityWidth, entityHeight := entityBounds(item, prefabInfoForEntity(catalog, item))
			if entityLeft < left || entityTop < top || entityLeft+entityWidth > right || entityTop+entityHeight > bottom {
				continue
			}
			state.Entities = append(state.Entities, editorcomponent.MoveEntitySelection{
				SourceIndex: index,
				EntityID:    item.ID,
				Entity:      cloneEditorEntity(item),
				OffsetX:     item.X - minX*TileSize,
				OffsetY:     item.Y - minY*TileSize,
			})
		}
	}
	return len(state.Layers) > 0
}

func (s *EditorToolSystem) applyMoveSelection(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, state *editorcomponent.MoveSelectionState) bool {
	if meta == nil || state == nil || !state.Active || state.Width <= 0 || state.Height <= 0 {
		return false
	}
	layers := layerEntities(w)
	if len(layers) != len(state.Layers) {
		session.Status = "Room move cancelled; layer count changed"
		return false
	}
	for layerIndex, entity := range layers {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer == nil {
			continue
		}
		captured := state.Layers[layerIndex]
		for y := 0; y < state.Height; y++ {
			for x := 0; x < state.Width; x++ {
				sourceIndex := cellIndex(meta, state.SourceMinX+x, state.SourceMinY+y)
				layer.Tiles[sourceIndex] = 0
				layer.TilesetUsage[sourceIndex] = nil
			}
		}
		for y := 0; y < state.Height; y++ {
			for x := 0; x < state.Width; x++ {
				destIndex := cellIndex(meta, state.DestMinX+x, state.DestMinY+y)
				capturedIndex := y*state.Width + x
				layer.Tiles[destIndex] = captured.Tiles[capturedIndex]
				layer.TilesetUsage[destIndex] = cloneTileUsage(captured.TilesetUsage[capturedIndex])
			}
		}
		if autotileEnabled(w) {
			queueAutotileFullLayer(w, layerIndex)
		}
	}
	if _, entities, ok := entitiesState(w); ok && entities != nil {
		for _, moved := range state.Entities {
			index := moved.SourceIndex
			if resolved := moveEntityIndexByID(entities.Items, moved, index); resolved >= 0 {
				index = resolved
			}
			if index < 0 || index >= len(entities.Items) {
				continue
			}
			entities.Items[index].X = state.DestMinX*TileSize + moved.OffsetX
			entities.Items[index].Y = state.DestMinY*TileSize + moved.OffsetY
		}
	}
	return true
}

func moveEntityIndexByID(items []levels.Entity, moved editorcomponent.MoveEntitySelection, fallback int) int {
	if moved.EntityID != "" {
		for index := range items {
			if items[index].ID == moved.EntityID {
				return index
			}
		}
	}
	if fallback >= 0 && fallback < len(items) {
		return fallback
	}
	return -1
}

func moveSelectionContainsCell(state *editorcomponent.MoveSelectionState, cellX, cellY int) bool {
	if state == nil || !state.Active || state.Width <= 0 || state.Height <= 0 {
		return false
	}
	return cellX >= state.DestMinX && cellY >= state.DestMinY && cellX < state.DestMinX+state.Width && cellY < state.DestMinY+state.Height
}

func (s *EditorToolSystem) updateMoveDestination(meta *editorcomponent.LevelMeta, state *editorcomponent.MoveSelectionState, minX, minY int) {
	if meta == nil || state == nil {
		return
	}
	maxX := maxInt(0, meta.Width-state.Width)
	maxY := maxInt(0, meta.Height-state.Height)
	state.DestMinX = clampInt(minX, 0, maxX)
	state.DestMinY = clampInt(minY, 0, maxY)
}

func rectFromCells(x0, y0, x1, y1 int) (int, int, int, int) {
	minX, maxX := x0, x1
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := y0, y1
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	return minX, minY, maxX - minX + 1, maxY - minY + 1
}

func (s *EditorToolSystem) sampleTileAtCursor(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState) {
	_, layer, ok := layerAt(w, session.CurrentLayer)
	if !ok || layer == nil || !withinLevel(meta, pointer.CellX, pointer.CellY) {
		return
	}
	usage := layer.TilesetUsage[cellIndex(meta, pointer.CellX, pointer.CellY)]
	if usage == nil || usage.Path == "" {
		return
	}
	session.SelectedTile = modelSelectionFromUsage(usage)
	if autotileEnabled(w) {
		session.SelectedTile.Index = 0
	}
	session.Status = "Sampled tile " + usage.Path
}

func (s *EditorToolSystem) applySpikeAt(w *ecs.World, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, cellX, cellY int) bool {
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return false
	}
	rotation := spikeRotationForCell(w, meta, cellX, cellY)
	x := cellX * TileSize
	y := cellY * TileSize
	for index := range entities.Items {
		item := &entities.Items[index]
		if !isSpikeEntity(*item) || item.X != x || item.Y != y {
			continue
		}
		props := ensureEntityProps(item)
		changed := false
		if item.Type != "spike" {
			item.Type = "spike"
			changed = true
		}
		if props["prefab"] != "spike.yaml" {
			props["prefab"] = "spike.yaml"
			changed = true
		}
		if currentLayer, ok := entityLayerIndex(props); !ok || currentLayer != session.CurrentLayer {
			props["layer"] = session.CurrentLayer
			changed = true
		}
		if toFloat(props["rotation"]) != rotation {
			props["rotation"] = rotation
			changed = true
		}
		return changed
	}
	entities.Items = append(entities.Items, levels.Entity{
		Type: "spike",
		X:    x,
		Y:    y,
		Props: map[string]interface{}{
			"layer":    session.CurrentLayer,
			"prefab":   "spike.yaml",
			"rotation": rotation,
		},
	})
	return true
}

func updateIdlePreview(stroke *editorcomponent.ToolStroke, session *editorcomponent.EditorSession, pointer *editorcomponent.PointerState) {
	if stroke == nil || session == nil || stroke.Active {
		return
	}
	if !pointer.HasCell {
		stroke.Preview = nil
		return
	}
	stroke.Preview = []editorcomponent.GridCell{{X: pointer.CellX, Y: pointer.CellY}}
}

func selectedTileValue(w *ecs.World, session *editorcomponent.EditorSession) (int, *levels.TileInfo) {
	selection := session.SelectedTile.Normalize()
	if autotileEnabled(w) {
		selection.Index = 0
		usage := selection.ToTileInfo()
		if usage != nil {
			usage.Auto = true
			usage.BaseIndex = 0
			usage.Mask = 0
		}
		return 0, usage
	}
	return selection.Index, selection.ToTileInfo()
}

func cloneTileUsage(info *levels.TileInfo) *levels.TileInfo {
	if info == nil {
		return nil
	}
	copied := *info
	return &copied
}

func tileUsageEqual(left, right *levels.TileInfo) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Path == right.Path && left.Index == right.Index && left.TileW == right.TileW && left.TileH == right.TileH && left.Auto == right.Auto && left.BaseIndex == right.BaseIndex && left.Mask == right.Mask
}

func bresenhamCells(x0, y0, x1, y1 int) []editorcomponent.GridCell {
	cells := make([]editorcomponent.GridCell, 0)
	dx := absInt(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInt(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		cells = append(cells, editorcomponent.GridCell{X: x0, Y: y0})
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
	return cells
}

func filledRectCells(x0, y0, x1, y1 int) []editorcomponent.GridCell {
	minX, maxX := x0, x1
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := y0, y1
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	cells := make([]editorcomponent.GridCell, 0, (maxX-minX+1)*(maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			cells = append(cells, editorcomponent.GridCell{X: x, Y: y})
		}
	}
	return cells
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func modelSelectionFromUsage(usage *levels.TileInfo) model.TileSelection {
	if usage == nil {
		return model.TileSelection{}
	}
	return model.TileSelection{Path: usage.Path, Index: usage.Index, TileW: usage.TileW, TileH: usage.TileH}.Normalize()
}

var _ ecs.System = (*EditorToolSystem)(nil)
