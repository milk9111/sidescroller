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
	_, placement, _ := prefabPlacementState(w)
	_, selection, _ := entitySelectionState(w)

	clampCurrentLayer(w, session)
	updateIdlePreview(stroke, session, pointer)
	if placement != nil && placement.SelectedPath != "" {
		if input.LeftJustReleased {
			stroke.Active = false
			stroke.Preview = nil
		}
		return
	}
	if selection != nil && (selection.Dragging || (selection.HoveredIndex >= 0 && input.LeftJustPressed)) {
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
	case editorcomponent.ToolLine:
		s.updateLineStroke(w, session, meta, pointer, input, stroke)
	case editorcomponent.ToolSpike:
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
