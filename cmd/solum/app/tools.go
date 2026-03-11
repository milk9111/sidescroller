package app

import (
	"container/list"
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"

	coreautotile "github.com/milk9111/sidescroller/internal/editorcore/autotile"
	coremodel "github.com/milk9111/sidescroller/internal/editorcore/model"
	"github.com/milk9111/sidescroller/levels"
)

const tileSize = coremodel.DefaultTileSize

type ViewportInput struct {
	MouseX             int
	MouseY             int
	WheelY             float64
	LeftDown           bool
	RightDown          bool
	MiddleDown         bool
	LeftJustPressed    bool
	LeftJustReleased   bool
	RightJustPressed   bool
	RightJustReleased  bool
	MiddleJustPressed  bool
	MiddleJustReleased bool
	Hovered            bool
}

func (s *State) selectTool(tool ToolKind) {
	switch tool {
	case ToolBrush, ToolErase, ToolFill, ToolBox, ToolBoxErase, ToolLine:
		s.ActiveTool = tool
		s.Status = fmt.Sprintf("Selected %s tool", tool)
	default:
		s.Status = "Select a valid tool"
	}
}

func (s *State) selectAsset(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		s.Status = "Select a valid asset"
		return
	}
	for _, asset := range s.Assets {
		if asset.Name == path || asset.Relative == path {
			s.SelectedTile.Path = asset.Name
			s.SelectedTile = s.SelectedTile.Normalize()
			s.TileIndexInput = fmt.Sprintf("%d", s.SelectedTile.Index)
			s.Status = "Selected tile asset " + asset.Name
			return
		}
	}
	s.Status = "Select a valid asset"
}

func (s *State) setSelectedTileIndex(index int) {
	if index < 0 {
		s.Status = "Tile index must be zero or greater"
		return
	}
	s.SelectedTile.Index = index
	s.SelectedTile = s.SelectedTile.Normalize()
	s.TileIndexInput = fmt.Sprintf("%d", s.SelectedTile.Index)
	s.Status = fmt.Sprintf("Selected tile index %d", s.SelectedTile.Index)
}

func (s *State) toggleAutotile() {
	s.Autotile.Enabled = !s.Autotile.Enabled
	if s.Autotile.Enabled {
		s.SelectedTile.Index = 0
		s.TileIndexInput = "0"
		for layerIndex := range s.Document.Layers {
			s.queueAutotileFullLayer(layerIndex)
		}
		s.flushAutotile()
		s.Status = "Autotile enabled"
		return
	}
	s.clearAutotileQueues()
	s.Status = "Autotile disabled"
}

func (s *State) SetTileIndexFromInput(raw string) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		s.Status = "Tile index must be an integer"
		return
	}
	s.Apply(Action{Kind: ActionSetTileIndex, Index: value})
}

func (s *State) UpdateViewport(rect image.Rectangle, input ViewportInput) {
	if s == nil {
		return
	}
	s.Camera.CanvasX = float64(rect.Min.X)
	s.Camera.CanvasY = float64(rect.Min.Y)
	s.Camera.CanvasW = float64(rect.Dx())
	s.Camera.CanvasH = float64(rect.Dy())
	if s.Camera.Zoom <= 0 {
		s.Camera.Zoom = 1
	}
	s.refreshPointer(rect, input)
	s.updateCamera(input)
	s.refreshPointer(rect, input)
	s.updateIdlePreview()
	s.updateToolInput(input)
	if input.RightJustPressed && s.Pointer.HasCell {
		s.sampleTileAtCursor()
	}
}

func (s *State) refreshPointer(rect image.Rectangle, input ViewportInput) {
	mouseX := float64(input.MouseX)
	mouseY := float64(input.MouseY)
	s.Pointer.OverCanvas = mouseX >= float64(rect.Min.X) && mouseX < float64(rect.Max.X) && mouseY >= float64(rect.Min.Y) && mouseY < float64(rect.Max.Y)
	s.Pointer.InCanvas = s.Pointer.OverCanvas
	s.Pointer.OverUI = !s.Pointer.OverCanvas
	s.Pointer.WorldX = s.Camera.X + (mouseX-float64(rect.Min.X))/s.Camera.Zoom
	s.Pointer.WorldY = s.Camera.Y + (mouseY-float64(rect.Min.Y))/s.Camera.Zoom
	s.Pointer.CellX = int(math.Floor(s.Pointer.WorldX / tileSize))
	s.Pointer.CellY = int(math.Floor(s.Pointer.WorldY / tileSize))
	s.Pointer.HasCell = s.Pointer.InCanvas && s.withinLevel(s.Pointer.CellX, s.Pointer.CellY)
}

func (s *State) updateCamera(input ViewportInput) {
	if input.MiddleDown && s.Pointer.InCanvas && !s.Camera.PanActive {
		s.Camera.PanActive = true
		s.Camera.PanMouseX = input.MouseX
		s.Camera.PanMouseY = input.MouseY
		s.Camera.PanStartX = s.Camera.X
		s.Camera.PanStartY = s.Camera.Y
	}
	if input.MiddleJustReleased || !input.MiddleDown {
		s.Camera.PanActive = false
	}
	if s.Camera.PanActive && input.MiddleDown && s.Pointer.InCanvas {
		dx := float64(input.MouseX-s.Camera.PanMouseX) / s.Camera.Zoom
		dy := float64(input.MouseY-s.Camera.PanMouseY) / s.Camera.Zoom
		s.Camera.X = s.Camera.PanStartX - dx
		s.Camera.Y = s.Camera.PanStartY - dy
	}
	if s.Pointer.InCanvas && input.WheelY != 0 {
		beforeWorldX := s.Pointer.WorldX
		beforeWorldY := s.Pointer.WorldY
		s.Camera.Zoom = clampFloat(s.Camera.Zoom+(input.WheelY*0.125), 0.25, 4.0)
		s.Camera.X = beforeWorldX - (float64(input.MouseX)-s.Camera.CanvasX)/s.Camera.Zoom
		s.Camera.Y = beforeWorldY - (float64(input.MouseY)-s.Camera.CanvasY)/s.Camera.Zoom
	}
	s.Camera.X = math.Max(s.Camera.X, -float64(tileSize)*2)
	s.Camera.Y = math.Max(s.Camera.Y, -float64(tileSize)*2)
}

func (s *State) updateToolInput(input ViewportInput) {
	if s.Camera.PanActive {
		return
	}
	if !s.Pointer.InCanvas && !s.ToolStroke.Active {
		return
	}
	switch s.ActiveTool {
	case ToolBrush:
		s.updatePaintStroke(input, false)
	case ToolErase:
		s.updatePaintStroke(input, true)
	case ToolFill:
		if input.LeftJustPressed && s.Pointer.HasCell {
			s.pushToolSnapshot("fill")
			if s.fillLayer(s.Pointer.CellX, s.Pointer.CellY) {
				s.finishToolSnapshot(true)
				s.Dirty = true
				s.Status = "Filled region"
			} else {
				s.finishToolSnapshot(false)
			}
		}
	case ToolBox:
		s.updateBoxStroke(input, false)
	case ToolBoxErase:
		s.updateBoxStroke(input, true)
	case ToolLine:
		s.updateLineStroke(input)
	}
}

func (s *State) updateIdlePreview() {
	if s.ToolStroke.Active {
		return
	}
	if !s.Pointer.HasCell {
		s.ToolStroke.Preview = nil
		return
	}
	s.ToolStroke.Preview = []GridCell{{X: s.Pointer.CellX, Y: s.Pointer.CellY}}
}

func (s *State) updatePaintStroke(input ViewportInput, erase bool) {
	if input.LeftJustPressed && s.Pointer.HasCell {
		reason := string(s.ActiveTool)
		s.pushToolSnapshot(reason)
		s.ToolStroke.Active = true
		s.ToolStroke.Tool = s.ActiveTool
		s.ToolStroke.StartCellX = s.Pointer.CellX
		s.ToolStroke.StartCellY = s.Pointer.CellY
		s.ToolStroke.LastCellX = s.Pointer.CellX
		s.ToolStroke.LastCellY = s.Pointer.CellY
		s.ToolStroke.Touched = map[int]struct{}{}
		if s.applyLineToLayer(s.Pointer.CellX, s.Pointer.CellY, erase) {
			s.ToolStroke.Changed = true
			s.Dirty = true
		}
	}
	if s.ToolStroke.Active && input.LeftDown && s.Pointer.HasCell {
		if s.applyLineToLayer(s.Pointer.CellX, s.Pointer.CellY, erase) {
			s.ToolStroke.Changed = true
			s.Dirty = true
		}
	}
	if s.ToolStroke.Active && (input.LeftJustReleased || !input.LeftDown) {
		s.finishToolSnapshot(s.ToolStroke.Changed)
		s.ToolStroke.Active = false
		s.ToolStroke.Touched = nil
		if s.ToolStroke.Changed {
			if erase {
				s.Status = "Erased tiles"
			} else {
				s.Status = "Painted tiles"
			}
		}
	}
}

func (s *State) updateBoxStroke(input ViewportInput, erase bool) {
	if input.LeftJustPressed && s.Pointer.HasCell {
		action := "box"
		if erase {
			action = "box erase"
		}
		s.pushToolSnapshot(action)
		s.ToolStroke.Active = true
		s.ToolStroke.Tool = s.ActiveTool
		s.ToolStroke.StartCellX = s.Pointer.CellX
		s.ToolStroke.StartCellY = s.Pointer.CellY
		s.ToolStroke.LastCellX = s.Pointer.CellX
		s.ToolStroke.LastCellY = s.Pointer.CellY
		s.ToolStroke.Preview = filledRectCells(s.Pointer.CellX, s.Pointer.CellY, s.Pointer.CellX, s.Pointer.CellY)
		return
	}
	if s.ToolStroke.Active && input.LeftDown && s.Pointer.HasCell {
		s.ToolStroke.LastCellX = s.Pointer.CellX
		s.ToolStroke.LastCellY = s.Pointer.CellY
		s.ToolStroke.Preview = filledRectCells(s.ToolStroke.StartCellX, s.ToolStroke.StartCellY, s.Pointer.CellX, s.Pointer.CellY)
	}
	if s.ToolStroke.Active && (input.LeftJustReleased || !input.LeftDown) {
		changed := false
		for _, cell := range s.ToolStroke.Preview {
			if !s.withinLevel(cell.X, cell.Y) {
				continue
			}
			if s.applyTileAt(cell.X, cell.Y, erase) {
				changed = true
			}
		}
		s.finishToolSnapshot(changed)
		s.ToolStroke.Active = false
		s.ToolStroke.Preview = nil
		if changed {
			s.Dirty = true
			if erase {
				s.Status = "Box erased"
			} else {
				s.Status = "Box placed"
			}
		}
	}
}

func (s *State) updateLineStroke(input ViewportInput) {
	if input.LeftJustPressed && s.Pointer.HasCell {
		s.pushToolSnapshot("line")
		s.ToolStroke.Active = true
		s.ToolStroke.Tool = ToolLine
		s.ToolStroke.StartCellX = s.Pointer.CellX
		s.ToolStroke.StartCellY = s.Pointer.CellY
		s.ToolStroke.LastCellX = s.Pointer.CellX
		s.ToolStroke.LastCellY = s.Pointer.CellY
		s.ToolStroke.Preview = bresenhamCells(s.Pointer.CellX, s.Pointer.CellY, s.Pointer.CellX, s.Pointer.CellY)
		return
	}
	if s.ToolStroke.Active && input.LeftDown && s.Pointer.HasCell {
		s.ToolStroke.LastCellX = s.Pointer.CellX
		s.ToolStroke.LastCellY = s.Pointer.CellY
		s.ToolStroke.Preview = bresenhamCells(s.ToolStroke.StartCellX, s.ToolStroke.StartCellY, s.Pointer.CellX, s.Pointer.CellY)
	}
	if s.ToolStroke.Active && (input.LeftJustReleased || !input.LeftDown) {
		changed := false
		for _, cell := range s.ToolStroke.Preview {
			if !s.withinLevel(cell.X, cell.Y) {
				continue
			}
			if s.applyTileAt(cell.X, cell.Y, false) {
				changed = true
			}
		}
		s.finishToolSnapshot(changed)
		s.ToolStroke.Active = false
		s.ToolStroke.Preview = nil
		if changed {
			s.Dirty = true
			s.Status = "Line placed"
		}
	}
}

func (s *State) pushToolSnapshot(reason string) {
	if s == nil {
		return
	}
	s.ToolStroke.SnapshotLen = len(s.UndoStack)
	s.ToolStroke.Changed = false
	s.pushSnapshot(reason)
}

func (s *State) finishToolSnapshot(changed bool) {
	if s == nil {
		return
	}
	if !changed && s.ToolStroke.SnapshotLen >= 0 && s.ToolStroke.SnapshotLen <= len(s.UndoStack) {
		s.UndoStack = s.UndoStack[:s.ToolStroke.SnapshotLen]
	}
	s.flushAutotile()
	s.ToolStroke.SnapshotLen = 0
	s.ToolStroke.Changed = false
}

func (s *State) applyLineToLayer(cellX, cellY int, erase bool) bool {
	changed := false
	for _, cell := range bresenhamCells(s.ToolStroke.LastCellX, s.ToolStroke.LastCellY, cellX, cellY) {
		if !s.withinLevel(cell.X, cell.Y) {
			continue
		}
		index := s.cellIndex(cell.X, cell.Y)
		if _, exists := s.ToolStroke.Touched[index]; exists {
			continue
		}
		if s.applyTileAt(cell.X, cell.Y, erase) {
			s.ToolStroke.Touched[index] = struct{}{}
			changed = true
		}
	}
	s.ToolStroke.LastCellX = cellX
	s.ToolStroke.LastCellY = cellY
	return changed
}

func (s *State) applyTileAt(cellX, cellY int, erase bool) bool {
	layer := s.currentLayerData()
	if layer == nil || !s.withinLevel(cellX, cellY) {
		return false
	}
	index := s.cellIndex(cellX, cellY)
	if erase {
		if layer.Tiles[index] == 0 && layer.TilesetUsage[index] == nil {
			return false
		}
		layer.Tiles[index] = 0
		layer.TilesetUsage[index] = nil
		if s.Autotile.Enabled {
			s.queueAutotileNeighborhood(s.CurrentLayer, cellX, cellY)
		}
		return true
	}
	newValue, newUsage := s.selectedTileValue()
	if strings.TrimSpace(s.SelectedTile.Path) == "" {
		s.Status = "No tileset selected"
		return false
	}
	if layer.Tiles[index] == newValue && tileUsageEqual(layer.TilesetUsage[index], newUsage) {
		return false
	}
	layer.Tiles[index] = newValue
	layer.TilesetUsage[index] = cloneTileUsage(newUsage)
	if s.Autotile.Enabled {
		s.queueAutotileNeighborhood(s.CurrentLayer, cellX, cellY)
	}
	return true
}

func (s *State) fillLayer(startX, startY int) bool {
	if s.Autotile.Enabled {
		return s.fillAutotileLayer(startX, startY)
	}
	return s.fillRawLayer(startX, startY)
}

func (s *State) fillRawLayer(startX, startY int) bool {
	layer := s.currentLayerData()
	if layer == nil || !s.withinLevel(startX, startY) {
		return false
	}
	startIndex := s.cellIndex(startX, startY)
	targetValue := layer.Tiles[startIndex]
	newValue, newUsage := s.selectedTileValue()
	if targetValue == newValue {
		return false
	}
	queue := list.New()
	queue.PushBack(GridCell{X: startX, Y: startY})
	visited := map[int]struct{}{startIndex: {}}
	changed := false
	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(GridCell)
		if !s.withinLevel(current.X, current.Y) {
			continue
		}
		index := s.cellIndex(current.X, current.Y)
		if layer.Tiles[index] != targetValue {
			continue
		}
		layer.Tiles[index] = newValue
		layer.TilesetUsage[index] = cloneTileUsage(newUsage)
		changed = true
		for _, next := range []GridCell{{X: current.X + 1, Y: current.Y}, {X: current.X - 1, Y: current.Y}, {X: current.X, Y: current.Y + 1}, {X: current.X, Y: current.Y - 1}} {
			if !s.withinLevel(next.X, next.Y) {
				continue
			}
			nextIndex := s.cellIndex(next.X, next.Y)
			if _, seen := visited[nextIndex]; seen {
				continue
			}
			visited[nextIndex] = struct{}{}
			queue.PushBack(next)
		}
	}
	return changed
}

func (s *State) fillAutotileLayer(startX, startY int) bool {
	layer := s.currentLayerData()
	if layer == nil || !s.withinLevel(startX, startY) {
		return false
	}
	startIndex := s.cellIndex(startX, startY)
	targetUsage := cloneTileUsage(layer.TilesetUsage[startIndex])
	_, newUsage := s.selectedTileValue()
	if newUsage == nil || autotileGroupEqual(targetUsage, newUsage) {
		return false
	}
	queue := list.New()
	queue.PushBack(GridCell{X: startX, Y: startY})
	visited := map[int]struct{}{startIndex: {}}
	changed := false
	for queue.Len() > 0 {
		current := queue.Remove(queue.Front()).(GridCell)
		if !s.withinLevel(current.X, current.Y) {
			continue
		}
		index := s.cellIndex(current.X, current.Y)
		if !autotileGroupEqual(layer.TilesetUsage[index], targetUsage) {
			continue
		}
		if !tileUsageEqual(layer.TilesetUsage[index], newUsage) {
			layer.TilesetUsage[index] = cloneTileUsage(newUsage)
			layer.Tiles[index] = newUsage.Index
			changed = true
		}
		for _, next := range []GridCell{{X: current.X + 1, Y: current.Y}, {X: current.X - 1, Y: current.Y}, {X: current.X, Y: current.Y + 1}, {X: current.X, Y: current.Y - 1}} {
			if !s.withinLevel(next.X, next.Y) {
				continue
			}
			nextIndex := s.cellIndex(next.X, next.Y)
			if _, seen := visited[nextIndex]; seen {
				continue
			}
			visited[nextIndex] = struct{}{}
			queue.PushBack(next)
		}
	}
	if changed {
		s.queueAutotileFullLayer(s.CurrentLayer)
	}
	return changed
}

func (s *State) sampleTileAtCursor() {
	layer := s.currentLayerData()
	if layer == nil || !s.withinLevel(s.Pointer.CellX, s.Pointer.CellY) {
		return
	}
	usage := layer.TilesetUsage[s.cellIndex(s.Pointer.CellX, s.Pointer.CellY)]
	if usage == nil || strings.TrimSpace(usage.Path) == "" {
		return
	}
	s.SelectedTile = coremodel.TileSelection{Path: usage.Path, Index: usage.Index, TileW: usage.TileW, TileH: usage.TileH}.Normalize()
	if s.Autotile.Enabled {
		s.SelectedTile.Index = 0
	}
	s.TileIndexInput = fmt.Sprintf("%d", s.SelectedTile.Index)
	s.Status = "Sampled tile " + usage.Path
}

func (s *State) selectedTileValue() (int, *levels.TileInfo) {
	selection := s.SelectedTile.Normalize()
	if s.Autotile.Enabled {
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

func (s *State) queueAutotileNeighborhood(layerIndex, cellX, cellY int) {
	if !s.Autotile.Enabled {
		return
	}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			nextX := cellX + dx
			nextY := cellY + dy
			if !s.withinLevel(nextX, nextY) {
				continue
			}
			s.queueAutotileCell(layerIndex, s.cellIndex(nextX, nextY))
		}
	}
}

func (s *State) queueAutotileCell(layerIndex, cellIndex int) {
	if s.Autotile.DirtyCells == nil {
		s.Autotile.DirtyCells = make(map[int]map[int]struct{})
	}
	if s.Autotile.DirtyCells[layerIndex] == nil {
		s.Autotile.DirtyCells[layerIndex] = make(map[int]struct{})
	}
	s.Autotile.DirtyCells[layerIndex][cellIndex] = struct{}{}
}

func (s *State) queueAutotileFullLayer(layerIndex int) {
	if s.Autotile.FullRebuild == nil {
		s.Autotile.FullRebuild = make(map[int]bool)
	}
	s.Autotile.FullRebuild[layerIndex] = true
}

func (s *State) clearAutotileQueues() {
	s.Autotile.DirtyCells = make(map[int]map[int]struct{})
	s.Autotile.FullRebuild = make(map[int]bool)
}

func (s *State) flushAutotile() {
	if !s.Autotile.Enabled {
		s.clearAutotileQueues()
		return
	}
	for layerIndex := range s.Autotile.FullRebuild {
		s.recomputeAutotileFullLayer(layerIndex)
	}
	for layerIndex, cells := range s.Autotile.DirtyCells {
		if s.Autotile.FullRebuild[layerIndex] {
			continue
		}
		for index := range cells {
			s.recomputeAutotileCell(layerIndex, index)
		}
	}
	s.clearAutotileQueues()
	s.syncDerivedState(false)
}

func (s *State) recomputeAutotileFullLayer(layerIndex int) {
	if layerIndex < 0 || layerIndex >= len(s.Document.Layers) {
		return
	}
	layer := &s.Document.Layers[layerIndex]
	for index, usage := range layer.TilesetUsage {
		if usage == nil || !usage.Auto {
			continue
		}
		s.applyComputedAutotileUsage(layer, index, usage)
	}
}

func (s *State) recomputeAutotileCell(layerIndex, index int) {
	if layerIndex < 0 || layerIndex >= len(s.Document.Layers) {
		return
	}
	layer := &s.Document.Layers[layerIndex]
	if index < 0 || index >= len(layer.TilesetUsage) {
		return
	}
	usage := layer.TilesetUsage[index]
	if usage == nil || !usage.Auto {
		return
	}
	s.applyComputedAutotileUsage(layer, index, usage)
}

func (s *State) applyComputedAutotileUsage(layer *coremodel.Layer, index int, usage *levels.TileInfo) {
	if layer == nil || usage == nil || !usage.Auto {
		return
	}
	cellX := index % s.Document.Width
	cellY := index / s.Document.Width
	mask := s.autotileMaskFor(layer, cellX, cellY, usage)
	offset, ok := coreautotile.ResolveOffset(mask, s.AutotileRemap)
	if !ok {
		offset = 0
	}
	usage.Mask = mask
	usage.Index = usage.BaseIndex + offset
	layer.Tiles[index] = usage.Index
}

func (s *State) autotileMaskFor(layer *coremodel.Layer, cellX, cellY int, usage *levels.TileInfo) uint8 {
	connected := func(x, y int) bool {
		if !s.withinLevel(x, y) {
			return true
		}
		neighbor := layer.TilesetUsage[s.cellIndex(x, y)]
		return neighbor != nil && neighbor.Auto && autotileGroupEqual(neighbor, usage)
	}
	north := connected(cellX, cellY-1)
	east := connected(cellX+1, cellY)
	south := connected(cellX, cellY+1)
	west := connected(cellX-1, cellY)
	return coreautotile.BuildMask(
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

func (s *State) currentLayerData() *coremodel.Layer {
	if s == nil || s.CurrentLayer < 0 || s.CurrentLayer >= len(s.Document.Layers) {
		return nil
	}
	return &s.Document.Layers[s.CurrentLayer]
}

func (s *State) withinLevel(cellX, cellY int) bool {
	return cellX >= 0 && cellY >= 0 && cellX < s.Document.Width && cellY < s.Document.Height
}

func (s *State) cellIndex(cellX, cellY int) int {
	return cellY*s.Document.Width + cellX
}

func clampFloat(value, minValue, maxValue float64) float64 {
	return math.Min(maxValue, math.Max(minValue, value))
}

func cloneTileUsage(info *levels.TileInfo) *levels.TileInfo {
	if info == nil {
		return nil
	}
	copy := *info
	return &copy
}

func tileUsageEqual(left, right *levels.TileInfo) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Path == right.Path && left.Index == right.Index && left.TileW == right.TileW && left.TileH == right.TileH && left.Auto == right.Auto && left.BaseIndex == right.BaseIndex && left.Mask == right.Mask
}

func autotileGroupEqual(left, right *levels.TileInfo) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Auto == right.Auto && left.Path == right.Path && left.BaseIndex == right.BaseIndex && left.TileW == right.TileW && left.TileH == right.TileH
}

func bresenhamCells(x0, y0, x1, y1 int) []GridCell {
	cells := make([]GridCell, 0)
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
		cells = append(cells, GridCell{X: x0, Y: y0})
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

func filledRectCells(x0, y0, x1, y1 int) []GridCell {
	minX, maxX := x0, x1
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := y0, y1
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	cells := make([]GridCell, 0, (maxX-minX+1)*(maxY-minY+1))
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			cells = append(cells, GridCell{X: x, Y: y})
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
