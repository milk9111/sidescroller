package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ebitenui/ebitenui"
	ebuiinput "github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	assetsPkg "github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/levels"
	prefabsPkg "github.com/milk9111/sidescroller/prefabs"
	"gopkg.in/yaml.v3"
)

// DummyLayer is a simple layer for demonstration.
type DummyLayer struct {
	Name         string
	Tiles        [][]int
	TilesetUsage [][]*levels.TileInfo
	Visible      bool
	Tint         color.RGBA
	Physics      bool
}

// EditorGame is the Ebiten game for the editor.
type Tool int

const (
	ToolBrush Tool = iota
	ToolErase
	ToolFill
	ToolLine
)

func (t Tool) String() string {
	switch t {
	case ToolBrush:
		return "Brush"
	case ToolErase:
		return "Erase"
	case ToolFill:
		return "Fill"
	case ToolLine:
		return "Line"
	default:
		return "Unknown"
	}
}

func promptLevelDimensions(defaultCols, defaultRows int) (int, int) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter level width in tiles (default %d): ", defaultCols)
	widthLine, _ := reader.ReadString('\n')
	widthLine = strings.TrimSpace(widthLine)
	cols := defaultCols
	if widthLine != "" {
		if v, err := strconv.Atoi(widthLine); err == nil && v > 0 {
			cols = v
		}
	}

	fmt.Printf("Enter level height in tiles (default %d): ", defaultRows)
	heightLine, _ := reader.ReadString('\n')
	heightLine = strings.TrimSpace(heightLine)
	rows := defaultRows
	if heightLine != "" {
		if v, err := strconv.Atoi(heightLine); err == nil && v > 0 {
			rows = v
		}
	}

	return cols, rows
}

type EditorGame struct {
	lineStart            *[2]int // nil if not started
	ui                   *ebitenui.UI
	gridSize             int
	gridWidth            int
	layers               []DummyLayer
	currentLayer         int
	tilesetZoom          *TilesetGridZoomable
	currentTool          Tool
	lastTool             Tool
	toolBar              *ToolBar
	layerPanel           *LayerPanel
	selectedTileset      *ebiten.Image
	selectedTileIndex    int
	selectedTilesetPath  string
	selectedPrefabImage  *ebiten.Image
	selectedPrefabName   string
	selectedPrefabPath   string
	selectedPrefabDrawW  int
	selectedPrefabDrawH  int
	zoom                 float64
	panX                 float64
	panY                 float64
	isPanning            bool
	lastPanX             int
	lastPanY             int
	gridPixel            *ebiten.Image
	leftPanelWidth       int
	rightPanelWidth      int
	gridRows             int
	gridCols             int
	showPhysicsHighlight bool
	autotileEnabled      bool
	undoStack            []editorSnapshot
	maxUndo              int
	isPainting           bool
	linePendingUndo      bool
	entities             []levels.Entity
	selectedEntity       int
	entityPendingUndo    bool
	entityDragging       bool
	transitionMode       bool
	transitionDragStart  *[2]int
	transitionDragEnd    *[2]int
	transitionDragTarget int
	transitionPropUndo   bool
	// legacy cache removed; kept for compatibility
	prefabImageCache        map[string]*ebiten.Image
	prefabMetaCache         map[string]prefabImageMeta
	savePath                string
	assetsDir               string
	fileNameInput           *widget.TextInput
	setTilesetSelect        func(tileIndex int)
	setTilesetSelectEnabled func(enabled bool)
	transitionUI            *TransitionUI
}

const (
	autoMaskN uint8 = 1 << iota
	autoMaskNE
	autoMaskE
	autoMaskSE
	autoMaskS
	autoMaskSW
	autoMaskW
	autoMaskNW
)

// 47-tile autotile mask order (ascending valid masks with corner constraints).
var auto47MaskOrder = []uint8{
	28, 124, 112, 16, 247, 223, 125, 31, 255, 241, 17, 253, 127, 95, 7, 199, 193, 1, 117, 87, 245, 4, 68, 64, 0, 213, 93, 215, 23, 209, 116, 92, 20, 84, 80, 29, 113, 197, 71, 21, 85, 81, 221, 119, 5, 69, 65,
}

var auto47MaskToIndex = initAuto47MaskToIndex()

var auto47TileIndices []int

func initAuto47MaskToIndex() []int {
	lookup := make([]int, 256)
	for i := range lookup {
		lookup[i] = -1
	}
	for idx, mask := range auto47MaskOrder {
		lookup[int(mask)] = idx
	}
	return lookup
}

func autoIndexForMask(mask uint8) int {
	if int(mask) >= len(auto47MaskToIndex) {
		return -1
	}
	return auto47MaskToIndex[int(mask)]
}

func autoTileIndexForMask(baseIndex int, mask uint8) int {
	idx := autoIndexForMask(mask)
	if idx < 0 {
		return baseIndex
	}
	if len(auto47TileIndices) == len(auto47MaskOrder) {
		mapped := auto47TileIndices[idx]
		if mapped >= 0 {
			return mapped
		}
	}
	return baseIndex + idx
}

type prefabImageMeta struct {
	Img   *ebiten.Image
	DrawW int
	DrawH int
}

type editorSnapshot struct {
	layers       []DummyLayer
	currentLayer int
	entities     []levels.Entity
}

func (g *EditorGame) TogglePhysicsForCurrentLayer() {
	if g.currentLayer < 0 || g.currentLayer >= len(g.layers) {
		return
	}
	g.pushUndo()
	g.layers[g.currentLayer].Physics = !g.layers[g.currentLayer].Physics
	g.updatePhysicsButtonLabel()
	g.syncTransitionUI()
}

func (g *EditorGame) ToggleAutotile() {
	g.autotileEnabled = !g.autotileEnabled
	if g.autotileEnabled {
		g.selectedTileIndex = 0
		if g.setTilesetSelect != nil {
			g.setTilesetSelect(0)
		}
	}
	if g.setTilesetSelectEnabled != nil {
		g.setTilesetSelectEnabled(!g.autotileEnabled)
	}
	g.updateAutotileButtonLabel()
}

func (g *EditorGame) updatePhysicsButtonLabel() {
	if g.layerPanel == nil {
		return
	}
	if g.currentLayer < 0 || g.currentLayer >= len(g.layers) {
		return
	}
	g.layerPanel.SetPhysicsButtonState(g.layers[g.currentLayer].Physics)
}

func (g *EditorGame) updateAutotileButtonLabel() {
	if g.layerPanel == nil {
		return
	}
	g.layerPanel.SetAutotileButtonState(g.autotileEnabled)
}

func (g *EditorGame) pushUndo() {
	if g.maxUndo <= 0 {
		g.maxUndo = 100
	}
	snapshot := editorSnapshot{
		layers:       cloneLayers(g.layers),
		currentLayer: g.currentLayer,
		entities:     cloneEntities(g.entities),
	}
	if len(g.undoStack) >= g.maxUndo {
		g.undoStack = g.undoStack[1:]
	}
	g.undoStack = append(g.undoStack, snapshot)
}

func (g *EditorGame) Undo() {
	if len(g.undoStack) == 0 {
		return
	}
	idx := len(g.undoStack) - 1
	snapshot := g.undoStack[idx]
	g.undoStack = g.undoStack[:idx]
	g.layers = cloneLayers(snapshot.layers)
	g.currentLayer = snapshot.currentLayer
	g.entities = cloneEntities(snapshot.entities)
	g.selectedEntity = -1
	g.entityDragging = false
	g.entityPendingUndo = false
	if g.layerPanel != nil {
		g.layerPanel.SetLayers(g.layerNames())
		g.layerPanel.SetSelected(g.currentLayer)
	}
	g.updatePhysicsButtonLabel()
}

func cloneLayers(src []DummyLayer) []DummyLayer {
	if src == nil {
		return nil
	}
	res := make([]DummyLayer, len(src))
	for i, layer := range src {
		res[i].Name = layer.Name
		res[i].Visible = layer.Visible
		res[i].Tint = layer.Tint
		res[i].Physics = layer.Physics
		if layer.Tiles != nil {
			res[i].Tiles = make([][]int, len(layer.Tiles))
			for y := range layer.Tiles {
				res[i].Tiles[y] = make([]int, len(layer.Tiles[y]))
				copy(res[i].Tiles[y], layer.Tiles[y])
			}
		}
		if layer.TilesetUsage != nil {
			res[i].TilesetUsage = make([][]*levels.TileInfo, len(layer.TilesetUsage))
			for y := range layer.TilesetUsage {
				res[i].TilesetUsage[y] = make([]*levels.TileInfo, len(layer.TilesetUsage[y]))
				for x := range layer.TilesetUsage[y] {
					if layer.TilesetUsage[y][x] != nil {
						copyInfo := *layer.TilesetUsage[y][x]
						res[i].TilesetUsage[y][x] = &copyInfo
					}
				}
			}
		}
	}
	return res
}

func cloneEntities(src []levels.Entity) []levels.Entity {
	if src == nil {
		return nil
	}
	res := make([]levels.Entity, len(src))
	for i, e := range src {
		res[i] = levels.Entity{Type: e.Type, X: e.X, Y: e.Y}
		if e.Props != nil {
			props := make(map[string]interface{}, len(e.Props))
			for k, v := range e.Props {
				props[k] = v
			}
			res[i].Props = props
		}
	}
	return res
}

func (g *EditorGame) entityIndexAtCell(cellX, cellY int) int {
	// Transitions are rectangles; allow selecting by clicking any covered cell.
	if idx := g.transitionIndexAtCell(cellX, cellY); idx >= 0 {
		return idx
	}
	for i := range g.entities {
		if strings.EqualFold(g.entities[i].Type, "transition") {
			continue
		}
		ex := g.entities[i].X / g.gridSize
		ey := g.entities[i].Y / g.gridSize
		if ex == cellX && ey == cellY {
			return i
		}
	}
	return -1
}

func (g *EditorGame) isTransitionEntity(idx int) bool {
	if g == nil || idx < 0 || idx >= len(g.entities) {
		return false
	}
	return strings.EqualFold(g.entities[idx].Type, "transition")
}

func (g *EditorGame) transitionIndexAtCell(cellX, cellY int) int {
	if g == nil {
		return -1
	}
	for i := range g.entities {
		if !strings.EqualFold(g.entities[i].Type, "transition") {
			continue
		}
		x0, y0, x1, y1 := g.transitionRectCells(g.entities[i])
		if cellX >= x0 && cellX <= x1 && cellY >= y0 && cellY <= y1 {
			return i
		}
	}
	return -1
}

func (g *EditorGame) transitionRectCells(ent levels.Entity) (x0, y0, x1, y1 int) {
	x0 = ent.X / g.gridSize
	y0 = ent.Y / g.gridSize
	w := float64(g.gridSize)
	h := float64(g.gridSize)
	if ent.Props != nil {
		if v, ok := ent.Props["w"]; ok {
			switch n := v.(type) {
			case float64:
				w = n
			case int:
				w = float64(n)
			}
		}
		if v, ok := ent.Props["h"]; ok {
			switch n := v.(type) {
			case float64:
				h = n
			case int:
				h = float64(n)
			}
		}
	}
	if w < float64(g.gridSize) {
		w = float64(g.gridSize)
	}
	if h < float64(g.gridSize) {
		h = float64(g.gridSize)
	}
	cols := int((w-1)/float64(g.gridSize)) + 1
	rows := int((h-1)/float64(g.gridSize)) + 1
	x1 = x0 + cols - 1
	y1 = y0 + rows - 1
	return
}

func (g *EditorGame) syncTransitionUI() {
	if g == nil || g.transitionUI == nil {
		return
	}
	g.transitionUI.SetMode(g.transitionMode)
	if g.selectedEntity < 0 || g.selectedEntity >= len(g.entities) || !g.isTransitionEntity(g.selectedEntity) {
		g.transitionUI.SetFormVisible(false)
		return
	}
	ent := g.entities[g.selectedEntity]
	props := ent.Props
	get := func(k string) string {
		if props == nil {
			return ""
		}
		if v, ok := props[k]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	g.transitionUI.SetFormVisible(true)
	g.transitionUI.SetFields(get("id"), get("to_level"), get("linked_id"), get("enter_dir"))
	g.transitionPropUndo = true
}

func (g *EditorGame) nextTransitionID() string {
	used := map[string]bool{}
	for i := range g.entities {
		if !strings.EqualFold(g.entities[i].Type, "transition") {
			continue
		}
		if g.entities[i].Props == nil {
			continue
		}
		if id, ok := g.entities[i].Props["id"].(string); ok && id != "" {
			used[id] = true
		}
	}
	for n := 1; n < 100000; n++ {
		id := fmt.Sprintf("t%d", n)
		if !used[id] {
			return id
		}
	}
	return "t"
}

func (g *EditorGame) finishTransitionDrag(endCellX, endCellY int) {
	if g == nil || g.transitionDragStart == nil {
		return
	}
	sx0, sy0 := g.transitionDragStart[0], g.transitionDragStart[1]
	sx1, sy1 := endCellX, endCellY
	if sx0 > sx1 {
		sx0, sx1 = sx1, sx0
	}
	if sy0 > sy1 {
		sy0, sy1 = sy1, sy0
	}

	px := sx0 * g.gridSize
	py := sy0 * g.gridSize
	w := (sx1 - sx0 + 1) * g.gridSize
	h := (sy1 - sy0 + 1) * g.gridSize
	if w < g.gridSize {
		w = g.gridSize
	}
	if h < g.gridSize {
		h = g.gridSize
	}

	// Resize existing if we started inside a transition; otherwise create new.
	if g.transitionDragTarget >= 0 && g.isTransitionEntity(g.transitionDragTarget) {
		g.pushUndo()
		ent := g.entities[g.transitionDragTarget]
		ent.Type = "transition"
		ent.X = px
		ent.Y = py
		if ent.Props == nil {
			ent.Props = map[string]interface{}{}
		}
		ent.Props["w"] = float64(w)
		ent.Props["h"] = float64(h)
		g.entities[g.transitionDragTarget] = ent
		g.selectedEntity = g.transitionDragTarget
		g.syncTransitionUI()
		return
	}

	g.pushUndo()
	props := map[string]interface{}{
		"id":        g.nextTransitionID(),
		"to_level":  "",
		"linked_id": "",
		"enter_dir": "",
		"w":         float64(w),
		"h":         float64(h),
	}
	g.entities = append(g.entities, levels.Entity{Type: "transition", X: px, Y: py, Props: props})
	g.selectedEntity = len(g.entities) - 1
	g.syncTransitionUI()
}

func (g *EditorGame) currentTileInfo(tileIndex int) *levels.TileInfo {
	if g.selectedTilesetPath == "" {
		return nil
	}
	return &levels.TileInfo{
		Path:  g.selectedTilesetPath,
		Index: tileIndex,
		TileW: g.gridSize,
		TileH: g.gridSize,
	}
}

func (g *EditorGame) currentAutoTileInfo(baseIndex, index int, mask uint8) *levels.TileInfo {
	if g.selectedTilesetPath == "" {
		return nil
	}
	return &levels.TileInfo{
		Path:      g.selectedTilesetPath,
		Index:     index,
		TileW:     g.gridSize,
		TileH:     g.gridSize,
		Auto:      true,
		BaseIndex: baseIndex,
		Mask:      mask,
	}
}

func (g *EditorGame) tileInfoAt(layerIdx, x, y int) *levels.TileInfo {
	if layerIdx < 0 || layerIdx >= len(g.layers) {
		return nil
	}
	if y < 0 || y >= len(g.layers[layerIdx].TilesetUsage) || x < 0 || x >= len(g.layers[layerIdx].TilesetUsage[y]) {
		return nil
	}
	return g.layers[layerIdx].TilesetUsage[y][x]
}

type tileFillKey struct {
	auto bool
	path string
	base int
	val  int
}

func (g *EditorGame) fillKeyAt(layerIdx, x, y int) tileFillKey {
	key := tileFillKey{}
	if layerIdx < 0 || layerIdx >= len(g.layers) {
		return key
	}
	if y < 0 || y >= len(g.layers[layerIdx].Tiles) || x < 0 || x >= len(g.layers[layerIdx].Tiles[y]) {
		return key
	}
	info := g.tileInfoAt(layerIdx, x, y)
	if info != nil && info.Auto {
		key.auto = true
		key.path = info.Path
		key.base = info.BaseIndex
		return key
	}
	key.auto = false
	key.val = g.layers[layerIdx].Tiles[y][x]
	return key
}

func (g *EditorGame) isAutoNeighbor(layerIdx, x, y, baseIndex int, path string) bool {
	info := g.tileInfoAt(layerIdx, x, y)
	if info == nil || !info.Auto {
		return false
	}
	if info.Path != path || info.BaseIndex != baseIndex {
		return false
	}
	if layerIdx < 0 || layerIdx >= len(g.layers) {
		return false
	}
	if y < 0 || y >= len(g.layers[layerIdx].Tiles) || x < 0 || x >= len(g.layers[layerIdx].Tiles[y]) {
		return false
	}
	return g.layers[layerIdx].Tiles[y][x] > 0
}

func (g *EditorGame) computeAutoMask(layerIdx, x, y, baseIndex int, path string) uint8 {
	var mask uint8
	n := g.isAutoNeighbor(layerIdx, x, y-1, baseIndex, path)
	e := g.isAutoNeighbor(layerIdx, x+1, y, baseIndex, path)
	s := g.isAutoNeighbor(layerIdx, x, y+1, baseIndex, path)
	w := g.isAutoNeighbor(layerIdx, x-1, y, baseIndex, path)
	if n {
		mask |= autoMaskN
	}
	if e {
		mask |= autoMaskE
	}
	if s {
		mask |= autoMaskS
	}
	if w {
		mask |= autoMaskW
	}
	if n && e && g.isAutoNeighbor(layerIdx, x+1, y-1, baseIndex, path) {
		mask |= autoMaskNE
	}
	if s && e && g.isAutoNeighbor(layerIdx, x+1, y+1, baseIndex, path) {
		mask |= autoMaskSE
	}
	if s && w && g.isAutoNeighbor(layerIdx, x-1, y+1, baseIndex, path) {
		mask |= autoMaskSW
	}
	if n && w && g.isAutoNeighbor(layerIdx, x-1, y-1, baseIndex, path) {
		mask |= autoMaskNW
	}
	return mask
}

func (g *EditorGame) setTileWithInfo(layerIdx, x, y, value int, info *levels.TileInfo) {
	if layerIdx < 0 || layerIdx >= len(g.layers) {
		return
	}
	if y < 0 || y >= len(g.layers[layerIdx].Tiles) || x < 0 || x >= len(g.layers[layerIdx].Tiles[y]) {
		return
	}
	g.layers[layerIdx].Tiles[y][x] = value
	if g.layers[layerIdx].TilesetUsage == nil || y >= len(g.layers[layerIdx].TilesetUsage) || x >= len(g.layers[layerIdx].TilesetUsage[y]) {
		return
	}
	if value <= 0 {
		g.layers[layerIdx].TilesetUsage[y][x] = nil
		return
	}
	if info == nil {
		info = g.currentTileInfo(value - 1)
	}
	g.layers[layerIdx].TilesetUsage[y][x] = info
}

func (g *EditorGame) setTile(layerIdx, x, y, value int) {
	g.setTileWithInfo(layerIdx, x, y, value, nil)
}

func (g *EditorGame) setAutoTile(layerIdx, x, y, baseIndex int) {
	if baseIndex < 0 {
		return
	}
	if !g.autotileEnabled {
		g.setTileWithInfo(layerIdx, x, y, baseIndex+1, nil)
		return
	}
	mask := uint8(0)
	index := autoTileIndexForMask(baseIndex, mask)
	info := g.currentAutoTileInfo(baseIndex, index, mask)
	if info == nil {
		return
	}
	g.setTileWithInfo(layerIdx, x, y, index+1, info)
	g.updateAutoTilesAround(layerIdx, x, y)
}

func (g *EditorGame) eraseTile(layerIdx, x, y int) {
	g.setTileWithInfo(layerIdx, x, y, 0, nil)
	g.updateAutoTilesAround(layerIdx, x, y)
}

func (g *EditorGame) recomputeAutoTileAt(layerIdx, x, y int) {
	if layerIdx < 0 || layerIdx >= len(g.layers) {
		return
	}
	if y < 0 || y >= len(g.layers[layerIdx].Tiles) || x < 0 || x >= len(g.layers[layerIdx].Tiles[y]) {
		return
	}
	info := g.tileInfoAt(layerIdx, x, y)
	if info == nil || !info.Auto {
		return
	}
	mask := g.computeAutoMask(layerIdx, x, y, info.BaseIndex, info.Path)
	index := autoTileIndexForMask(info.BaseIndex, mask)
	info.Index = index
	info.Mask = mask
	g.setTileWithInfo(layerIdx, x, y, index+1, info)
}

func (g *EditorGame) updateAutoTilesAround(layerIdx, x, y int) {
	if !g.autotileEnabled {
		return
	}
	for yy := y - 1; yy <= y+1; yy++ {
		for xx := x - 1; xx <= x+1; xx++ {
			g.recomputeAutoTileAt(layerIdx, xx, yy)
		}
	}
}

func (g *EditorGame) updateAutoTilesInRegion(layerIdx, minX, minY, maxX, maxY int) {
	if !g.autotileEnabled {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			g.recomputeAutoTileAt(layerIdx, x, y)
		}
	}
}

func (g *EditorGame) updateAutoTilesForLayer(layerIdx int) {
	if !g.autotileEnabled {
		return
	}
	if layerIdx < 0 || layerIdx >= len(g.layers) {
		return
	}
	rows := len(g.layers[layerIdx].Tiles)
	if rows == 0 {
		return
	}
	cols := len(g.layers[layerIdx].Tiles[0])
	g.updateAutoTilesInRegion(layerIdx, 0, 0, cols-1, rows-1)
}

// floodFill fills contiguous tiles of the same value starting from (x, y)
func (g *EditorGame) floodFillAuto(x, y int, target tileFillKey, replacementBase int) {
	if g.currentLayer < 0 || g.currentLayer >= len(g.layers) {
		return
	}
	if y < 0 || y >= len(g.layers[g.currentLayer].Tiles) || x < 0 || x >= len(g.layers[g.currentLayer].Tiles[y]) {
		return
	}
	if g.fillKeyAt(g.currentLayer, x, y) != target {
		return
	}
	g.setAutoTile(g.currentLayer, x, y, replacementBase)
	g.floodFillAuto(x+1, y, target, replacementBase)
	g.floodFillAuto(x-1, y, target, replacementBase)
	g.floodFillAuto(x, y+1, target, replacementBase)
	g.floodFillAuto(x, y-1, target, replacementBase)
}

func (g *EditorGame) floodFillPlain(x, y, target, replacement int) {
	if target == replacement {
		return
	}
	if g.currentLayer < 0 || g.currentLayer >= len(g.layers) {
		return
	}
	if y < 0 || y >= len(g.layers[g.currentLayer].Tiles) || x < 0 || x >= len(g.layers[g.currentLayer].Tiles[y]) {
		return
	}
	if g.layers[g.currentLayer].Tiles[y][x] != target {
		return
	}
	g.setTile(g.currentLayer, x, y, replacement)
	g.floodFillPlain(x+1, y, target, replacement)
	g.floodFillPlain(x-1, y, target, replacement)
	g.floodFillPlain(x, y+1, target, replacement)
	g.floodFillPlain(x, y-1, target, replacement)
}

func (g *EditorGame) layerNames() []string {
	names := make([]string, len(g.layers))
	for i, layer := range g.layers {
		if layer.Name != "" {
			names[i] = layer.Name
		} else {
			switch i {
			case 0:
				names[i] = "Background"
			case 1:
				names[i] = "Physics"
			default:
				names[i] = fmt.Sprintf("Layer %d", i)
			}
		}
	}
	return names
}

func (g *EditorGame) AddLayer() {
	if g.gridRows <= 0 || g.gridCols <= 0 {
		return
	}
	g.pushUndo()
	tiles := make([][]int, g.gridRows)
	for y := range tiles {
		tiles[y] = make([]int, g.gridCols)
	}
	name := fmt.Sprintf("Layer %d", len(g.layers))
	if len(g.layers) == 0 {
		name = "Background"
	} else if len(g.layers) == 1 {
		name = "Physics"
	}
	g.layers = append(g.layers, DummyLayer{
		Name:         name,
		Tiles:        tiles,
		TilesetUsage: make([][]*levels.TileInfo, g.gridRows),
		Visible:      true,
		Tint:         color.RGBA{R: 100, G: 200, B: 255, A: 255},
		Physics:      false,
	})
	for y := range g.layers[len(g.layers)-1].TilesetUsage {
		g.layers[len(g.layers)-1].TilesetUsage[y] = make([]*levels.TileInfo, g.gridCols)
	}
	g.currentLayer = len(g.layers) - 1
	if g.layerPanel != nil {
		g.layerPanel.SetLayers(g.layerNames())
		g.layerPanel.SetSelected(g.currentLayer)
	}
	g.updatePhysicsButtonLabel()
}

func (g *EditorGame) MoveLayerUp(idx int) {
	if idx < 0 || idx >= len(g.layers)-1 {
		return
	}
	g.pushUndo()
	g.layers[idx], g.layers[idx+1] = g.layers[idx+1], g.layers[idx]
	if g.currentLayer == idx {
		g.currentLayer = idx + 1
	} else if g.currentLayer == idx+1 {
		g.currentLayer = idx
	}
	if g.layerPanel != nil {
		g.layerPanel.SetLayers(g.layerNames())
		g.layerPanel.SetSelected(g.currentLayer)
	}
	g.updatePhysicsButtonLabel()
}

func (g *EditorGame) MoveLayerDown(idx int) {
	if idx <= 0 || idx >= len(g.layers) {
		return
	}
	g.pushUndo()
	g.layers[idx], g.layers[idx-1] = g.layers[idx-1], g.layers[idx]
	if g.currentLayer == idx {
		g.currentLayer = idx - 1
	} else if g.currentLayer == idx-1 {
		g.currentLayer = idx
	}
	if g.layerPanel != nil {
		g.layerPanel.SetLayers(g.layerNames())
		g.layerPanel.SetSelected(g.currentLayer)
	}
	g.updatePhysicsButtonLabel()
}

func (g *EditorGame) Update() error {
	// If the UI has a focused text widget (user is typing), suppress hotkeys.
	suppressHotkeys := false
	if g.ui != nil {
		if fw := g.ui.GetFocusedWidget(); fw != nil {
			switch fw.(type) {
			case *widget.TextInput:
				suppressHotkeys = true
			}
		}
	}

	if !suppressHotkeys {
		if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
			os.Exit(0)
		}

		// Cycle layers (Q/E)
		if inpututil.IsKeyJustPressed(ebiten.KeyQ) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
			if len(g.layers) > 0 {
				g.currentLayer--
				if g.currentLayer < 0 {
					g.currentLayer = len(g.layers) - 1
				}
				if g.layerPanel != nil {
					g.layerPanel.SetSelected(g.currentLayer)
				}
				g.updatePhysicsButtonLabel()
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyE) && !ebiten.IsKeyPressed(ebiten.KeyControl) {
			if len(g.layers) > 0 {
				g.currentLayer++
				if g.currentLayer >= len(g.layers) {
					g.currentLayer = 0
				}
				if g.layerPanel != nil {
					g.layerPanel.SetSelected(g.currentLayer)
				}
				g.updatePhysicsButtonLabel()
			}
		}

		// New layer (N)
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.AddLayer()
		}

		// Tool switching hotkeys
		if inpututil.IsKeyJustPressed(ebiten.KeyB) && ebiten.IsKeyPressed(ebiten.KeyControl) {
			g.currentTool = ToolBrush
			log.Println("Switched to Brush tool")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyE) && ebiten.IsKeyPressed(ebiten.KeyControl) {
			g.currentTool = ToolErase
			log.Println("Switched to Erase tool")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyF) && ebiten.IsKeyPressed(ebiten.KeyControl) {
			g.currentTool = ToolFill
			log.Println("Switched to Fill tool")
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyL) && ebiten.IsKeyPressed(ebiten.KeyControl) {
			g.currentTool = ToolLine
			log.Println("Switched to Line tool")
		}

		// Undo (Ctrl+Z)
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) && ebiten.IsKeyPressed(ebiten.KeyControl) {
			g.Undo()
		}

		if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if g.selectedEntity >= 0 && g.selectedEntity < len(g.entities) {
				g.pushUndo()
				g.entities = append(g.entities[:g.selectedEntity], g.entities[g.selectedEntity+1:]...)
				g.selectedEntity = -1
				g.entityDragging = false
				g.entityPendingUndo = false
				g.syncTransitionUI()
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.selectedPrefabName = ""
			g.selectedPrefabPath = ""
			g.selectedEntity = -1
			g.entityDragging = false
			g.entityPendingUndo = false
			g.syncTransitionUI()
		}

		// Save (Ctrl+S) - use filename in the left-panel input
		if inpututil.IsKeyJustPressed(ebiten.KeyS) && ebiten.IsKeyPressed(ebiten.KeyControl) {
			var name string
			if g.fileNameInput != nil {
				name = strings.TrimSpace(g.fileNameInput.GetText())
			}
			if name == "" {
				log.Println("No filename specified in File field; save aborted")
			} else {
				path := g.normalizeSavePath(name)
				if err := g.SaveLevelToPath(path); err != nil {
					log.Printf("Save failed: %v", err)
				} else {
					g.savePath = path
				}
			}
		}

		// Physics metadata hotkeys
		if inpututil.IsKeyJustPressed(ebiten.KeyH) {
			g.TogglePhysicsForCurrentLayer()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyY) {
			g.showPhysicsHighlight = !g.showPhysicsHighlight
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyT) {
			g.ToggleAutotile()
		}
	}

	if g.currentTool != g.lastTool {
		if g.toolBar != nil {
			g.toolBar.SetTool(g.currentTool)
		}
		g.lastTool = g.currentTool
	}

	if g.ui != nil {
		g.ui.Update()
	}
	// Handle pan (middle mouse drag)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		g.isPanning = true
		g.lastPanX, g.lastPanY = ebiten.CursorPosition()
	}
	if g.isPanning && ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		cx, cy := ebiten.CursorPosition()
		dx := cx - g.lastPanX
		dy := cy - g.lastPanY
		g.panX += float64(dx)
		g.panY += float64(dy)
		g.lastPanX, g.lastPanY = cx, cy
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonMiddle) {
		g.isPanning = false
	}

	// Handle zoom (mouse wheel, centered on cursor)
	if _, wy := ebiten.Wheel(); wy != 0 {
		cx, cy := ebiten.CursorPosition()
		oldZoom := g.zoom
		if wy > 0 {
			g.zoom *= 1.1
		} else {
			g.zoom /= 1.1
		}
		if g.zoom < 0.25 {
			g.zoom = 0.25
		}
		if g.zoom > 4.0 {
			g.zoom = 4.0
		}
		if g.zoom != oldZoom {
			worldX := (float64(cx) - g.panX) / oldZoom
			worldY := (float64(cy) - g.panY) / oldZoom
			g.panX = float64(cx) - worldX*g.zoom
			g.panY = float64(cy) - worldY*g.zoom
		}
	}

	// Mouse to grid mapping (screen -> world -> cell)
	sx, sy := ebiten.CursorPosition()
	screenW, _ := ebiten.Monitor().Size()
	if sx < g.leftPanelWidth || sy < 0 || sx >= g.rightPanelWidth+screenW {
		return nil
	}
	worldX := (float64(sx-g.leftPanelWidth) - g.panX) / g.zoom
	worldY := (float64(sy) - g.panY) / g.zoom
	if worldX < 0 || worldY < 0 {
		return nil
	}
	cellX := int(worldX) / g.gridSize
	cellY := int(worldY) / g.gridSize
	// Brush/Erase/Fill/Line tool logic
	if g.currentLayer < 0 || g.currentLayer >= len(g.layers) {
		return nil
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if g.transitionMode && g.transitionDragStart != nil {
			g.finishTransitionDrag(cellX, cellY)
		}
		g.isPainting = false
		g.linePendingUndo = false
		g.entityDragging = false
		g.entityPendingUndo = false
		g.transitionDragStart = nil
		g.transitionDragEnd = nil
		g.transitionDragTarget = -1
	}
	// If the UI is hovered, ignore left-click tool actions so toolbar/button clicks
	// don't also paint the tilemap underneath.
	if !ebuiinput.UIHovered {
		if cellY >= 0 && cellY < len(g.layers[g.currentLayer].Tiles) && cellX >= 0 && cellX < len(g.layers[g.currentLayer].Tiles[cellY]) {
			// Transition placement mode overrides all other interactions.
			if g.transitionMode {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					g.transitionDragStart = &[2]int{cellX, cellY}
					g.transitionDragEnd = &[2]int{cellX, cellY}
					g.transitionDragTarget = g.transitionIndexAtCell(cellX, cellY)
					if g.transitionDragTarget >= 0 {
						g.selectedEntity = g.transitionDragTarget
						g.syncTransitionUI()
					}
					// Suppress prefab placement/dragging while in transition mode.
					g.selectedPrefabName = ""
					g.selectedPrefabPath = ""
					g.entityDragging = false
					g.entityPendingUndo = false
					return nil
				}
				if g.transitionDragStart != nil && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
					g.transitionDragEnd = &[2]int{cellX, cellY}
					return nil
				}
			}

			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				entityIdx := g.entityIndexAtCell(cellX, cellY)
				if entityIdx >= 0 {
					g.selectedEntity = entityIdx
					g.syncTransitionUI()
					// Don't allow drag-moving transitions in normal mode.
					if !g.isTransitionEntity(entityIdx) {
						g.entityDragging = true
						g.entityPendingUndo = true
					}
					return nil
				}
				if g.selectedPrefabName != "" {
					g.pushUndo()
					var props map[string]interface{}
					if g.selectedPrefabPath != "" {
						props = map[string]interface{}{"prefab": g.selectedPrefabPath}
					}
					g.entities = append(g.entities, levels.Entity{
						Type:  g.selectedPrefabName,
						X:     cellX * g.gridSize,
						Y:     cellY * g.gridSize,
						Props: props,
					})
					g.selectedEntity = len(g.entities) - 1
					g.syncTransitionUI()
					g.entityDragging = false
					g.entityPendingUndo = false
					return nil
				}
			}
			if g.entityDragging && g.selectedEntity >= 0 && g.selectedEntity < len(g.entities) && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				if g.isTransitionEntity(g.selectedEntity) {
					return nil
				}
				if g.entityPendingUndo {
					g.pushUndo()
					g.entityPendingUndo = false
				}
				g.entities[g.selectedEntity].X = cellX * g.gridSize
				g.entities[g.selectedEntity].Y = cellY * g.gridSize
				return nil
			}
			if g.selectedPrefabName != "" {
				return nil
			}
			switch g.currentTool {
			case ToolBrush:
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					g.pushUndo()
					g.isPainting = true
				}
				if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.isPainting {
					if g.selectedTileIndex >= 0 {
						g.setAutoTile(g.currentLayer, cellX, cellY, g.selectedTileIndex)
					}
				}
			case ToolErase:
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					g.pushUndo()
					g.isPainting = true
				}
				if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.isPainting {
					g.eraseTile(g.currentLayer, cellX, cellY)
				}
			case ToolFill:
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					if g.selectedTileIndex >= 0 {
						g.pushUndo()
						if g.autotileEnabled {
							start := g.fillKeyAt(g.currentLayer, cellX, cellY)
							g.floodFillAuto(cellX, cellY, start, g.selectedTileIndex)
							g.updateAutoTilesForLayer(g.currentLayer)
						} else {
							start := g.layers[g.currentLayer].Tiles[cellY][cellX]
							replace := g.selectedTileIndex + 1
							g.floodFillPlain(cellX, cellY, start, replace)
						}
					}
				}
			case ToolLine:
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					// Set start point
					g.lineStart = &[2]int{cellX, cellY}
					g.linePendingUndo = true
				}
				if g.lineStart != nil && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
					// Set end point and draw line
					x0, y0 := g.lineStart[0], g.lineStart[1]
					x1, y1 := cellX, cellY
					if g.linePendingUndo && g.selectedTileIndex >= 0 {
						g.pushUndo()
						g.linePendingUndo = false
					}
					minX, maxX := x0, x1
					minY, maxY := y0, y1
					if minX > maxX {
						minX, maxX = maxX, minX
					}
					if minY > maxY {
						minY, maxY = maxY, minY
					}
					for _, pt := range bresenhamLine(x0, y0, x1, y1) {
						px, py := pt[0], pt[1]
						if py >= 0 && py < len(g.layers[g.currentLayer].Tiles) && px >= 0 && px < len(g.layers[g.currentLayer].Tiles[py]) {
							if g.selectedTileIndex >= 0 {
								g.setAutoTile(g.currentLayer, px, py, g.selectedTileIndex)
							}
						}
					}
					g.updateAutoTilesInRegion(g.currentLayer, minX-1, minY-1, maxX+1, maxY+1)
					g.lineStart = nil
				}
			}
		}
	}
	return nil
}

func (g *EditorGame) Draw(screen *ebiten.Image) {
	if g.gridPixel == nil {
		g.gridPixel = ebiten.NewImage(1, 1)
		g.gridPixel.Fill(color.White)
	}
	// Draw tiled layers (if visible)
	for li := range g.layers {
		layer := g.layers[li]
		if !layer.Visible {
			continue
		}
		for y, row := range layer.Tiles {
			for x, v := range row {
				if v == 0 {
					continue
				}
				if g.selectedTileset != nil {
					tileSize := g.gridSize
					tsW, tsH := g.selectedTileset.Size()
					tilesX := tsW / tileSize
					tileIndex := v - 1
					if tilesX > 0 && tileIndex >= 0 {
						tileX := tileIndex % tilesX
						tileY := tileIndex / tilesX
						if tileX*tileSize < tsW && tileY*tileSize < tsH {
							sub := g.selectedTileset.SubImage(
								image.Rect(tileX*tileSize, tileY*tileSize, (tileX+1)*tileSize, (tileY+1)*tileSize),
							).(*ebiten.Image)
							op := &ebiten.DrawImageOptions{}
							op.GeoM.Scale(g.zoom, g.zoom)
							op.GeoM.Translate(float64(x*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(y*g.gridSize)*g.zoom+g.panY)
							screen.DrawImage(sub, op)
							continue
						}
					}
				}
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
				op.GeoM.Translate(float64(x*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(y*g.gridSize)*g.zoom+g.panY)
				op.ColorScale.Scale(float32(layer.Tint.R)/255, float32(layer.Tint.G)/255, float32(layer.Tint.B)/255, 0.5)
				screen.DrawImage(g.gridPixel, op)
			}
		}
	}
	// Draw line preview
	if g.currentTool == ToolLine && g.lineStart != nil {
		cx, cy := ebiten.CursorPosition()
		screenW, _ := ebiten.Monitor().Size()
		if cx >= g.leftPanelWidth && cy >= 0 && cx < screenW+g.rightPanelWidth {
			worldX := (float64(cx-g.leftPanelWidth) - g.panX) / g.zoom
			worldY := (float64(cy) - g.panY) / g.zoom
			endX := int(worldX) / g.gridSize
			endY := int(worldY) / g.gridSize
			startX, startY := g.lineStart[0], g.lineStart[1]
			for _, pt := range bresenhamLine(startX, startY, endX, endY) {
				px, py := pt[0], pt[1]
				if g.currentLayer < 0 || g.currentLayer >= len(g.layers) || py < 0 || py >= len(g.layers[g.currentLayer].Tiles) || px < 0 || px >= len(g.layers[g.currentLayer].Tiles[py]) {
					continue
				}
				if g.selectedTileset != nil && g.selectedTileIndex >= 0 {
					tileSize := g.gridSize
					tsW, tsH := g.selectedTileset.Size()
					tilesX := tsW / tileSize
					tileIndex := g.selectedTileIndex
					if tilesX > 0 && tileIndex >= 0 {
						tileX := tileIndex % tilesX
						tileY := tileIndex / tilesX
						if tileX*tileSize < tsW && tileY*tileSize < tsH {
							sub := g.selectedTileset.SubImage(
								image.Rect(tileX*tileSize, tileY*tileSize, (tileX+1)*tileSize, (tileY+1)*tileSize),
							).(*ebiten.Image)
							op := &ebiten.DrawImageOptions{}
							op.GeoM.Scale(g.zoom, g.zoom)
							op.GeoM.Translate(float64(px*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(py*g.gridSize)*g.zoom+g.panY)
							op.ColorScale.Scale(1, 1, 1, 0.5)
							screen.DrawImage(sub, op)
							continue
						}
					}
				}
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
				op.GeoM.Translate(float64(px*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(py*g.gridSize)*g.zoom+g.panY)
				op.ColorScale.Scale(float32(g.layers[g.currentLayer].Tint.R)/255, float32(g.layers[g.currentLayer].Tint.G)/255, float32(g.layers[g.currentLayer].Tint.B)/255, 0.5)
				screen.DrawImage(g.gridPixel, op)
			}
		}
	}
	if g.showPhysicsHighlight {
		overlay := color.RGBA{R: 255, G: 80, B: 80, A: 120}
		for li := range g.layers {
			layer := g.layers[li]
			if !layer.Physics {
				continue
			}
			for y, row := range layer.Tiles {
				for x, v := range row {
					if v == 0 {
						continue
					}
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
					op.GeoM.Translate(float64(x*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(y*g.gridSize)*g.zoom+g.panY)
					op.ColorScale.Scale(float32(overlay.R)/255, float32(overlay.G)/255, float32(overlay.B)/255, float32(overlay.A)/255)
					screen.DrawImage(g.gridPixel, op)
				}
			}
		}
	}

	// Draw transitions as tile overlays (yellow; selected = green)
	for i := range g.entities {
		if !strings.EqualFold(g.entities[i].Type, "transition") {
			continue
		}
		ent := g.entities[i]
		x0, y0, x1, y1 := g.transitionRectCells(ent)
		marker := color.RGBA{R: 255, G: 230, B: 80, A: 180}
		if i == g.selectedEntity {
			marker = color.RGBA{R: 80, G: 255, B: 120, A: 200}
		}
		for ty := y0; ty <= y1; ty++ {
			for tx := x0; tx <= x1; tx++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
				op.GeoM.Translate(float64(tx*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(ty*g.gridSize)*g.zoom+g.panY)
				op.ColorScale.Scale(float32(marker.R)/255, float32(marker.G)/255, float32(marker.B)/255, float32(marker.A)/255)
				screen.DrawImage(g.gridPixel, op)
			}
		}
	}
	// Transition drag preview (blue)
	if g.transitionMode && g.transitionDragStart != nil && g.transitionDragEnd != nil {
		sx0, sy0 := g.transitionDragStart[0], g.transitionDragStart[1]
		sx1, sy1 := g.transitionDragEnd[0], g.transitionDragEnd[1]
		if sx0 > sx1 {
			sx0, sx1 = sx1, sx0
		}
		if sy0 > sy1 {
			sy0, sy1 = sy1, sy0
		}
		preview := color.RGBA{R: 80, G: 180, B: 255, A: 90}
		for ty := sy0; ty <= sy1; ty++ {
			for tx := sx0; tx <= sx1; tx++ {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
				op.GeoM.Translate(float64(tx*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(ty*g.gridSize)*g.zoom+g.panY)
				op.ColorScale.Scale(float32(preview.R)/255, float32(preview.G)/255, float32(preview.B)/255, float32(preview.A)/255)
				screen.DrawImage(g.gridPixel, op)
			}
		}
	}
	// Draw prefab entities
	for i := range g.entities {
		ent := g.entities[i]
		if strings.EqualFold(ent.Type, "transition") {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
		op.GeoM.Translate(float64(ent.X)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(ent.Y)*g.zoom+g.panY)
		marker := color.RGBA{R: 80, G: 220, B: 120, A: 200}
		if i == g.selectedEntity {
			marker = color.RGBA{R: 255, G: 220, B: 80, A: 220}
		}
		op.ColorScale.Scale(float32(marker.R)/255, float32(marker.G)/255, float32(marker.B)/255, float32(marker.A)/255)
		screen.DrawImage(g.gridPixel, op)
		// Try to draw a cached prefab image/meta for this entity, falling back to a colored square.
		var meta *prefabImageMeta
		if g.selectedPrefabName != "" && ent.Type == g.selectedPrefabName && g.selectedPrefabImage != nil {
			// Use currently-selected prefab preview image and size if available
			meta = &prefabImageMeta{Img: g.selectedPrefabImage, DrawW: g.selectedPrefabDrawW, DrawH: g.selectedPrefabDrawH}
		} else if ent.Props != nil {
			if p, ok := ent.Props["prefab"].(string); ok && p != "" {
				if g.prefabMetaCache == nil {
					g.prefabMetaCache = make(map[string]prefabImageMeta)
				}
				if cached, ok := g.prefabMetaCache[p]; ok {
					meta = &cached
				} else {
					// load prefab spec to determine draw size and image
					var pm prefabImageMeta
					data, err := prefabsPkg.Load(p)
					if err == nil {
						var spec struct {
							Animation *prefabsPkg.AnimationSpec `yaml:"animation"`
							Sprite    *prefabsPkg.SpriteSpec    `yaml:"sprite"`
						}
						if err := yaml.Unmarshal(data, &spec); err == nil {
							if spec.Animation != nil && len(spec.Animation.Defs) > 0 {
								// pick first def sorted
								keys := make([]string, 0, len(spec.Animation.Defs))
								for k := range spec.Animation.Defs {
									keys = append(keys, k)
								}
								sort.Strings(keys)
								def := spec.Animation.Defs[keys[0]]
								if sheet, err := assetsPkg.LoadImage(spec.Animation.Sheet); err == nil && sheet != nil {
									// extract subimage rectangle
									x := def.ColStart * def.FrameW
									y := def.Row * def.FrameH
									r := image.Rect(x, y, x+def.FrameW, y+def.FrameH)
									if sub := sheet.SubImage(r); sub != nil {
										if bi, ok := sub.(*ebiten.Image); ok {
											pm.Img = bi
											pm.DrawW = def.FrameW
											pm.DrawH = def.FrameH
										}
									}
								}
							}
							if pm.Img == nil && spec.Sprite != nil && spec.Sprite.Image != "" {
								if img, err := assetsPkg.LoadImage(spec.Sprite.Image); err == nil {
									pm.Img = img
									w, h := img.Size()
									pm.DrawW = w
									pm.DrawH = h
								}
							}
						}
					}
					if pm.Img == nil {
						pm.DrawW = g.gridSize
						pm.DrawH = g.gridSize
					}
					g.prefabMetaCache[p] = pm
					meta = &pm
				}
			}
		}
		if meta != nil && meta.Img != nil {
			img := meta.Img
			iw, ih := img.Size()
			scaleX := float64(meta.DrawW) / float64(iw)
			scaleY := float64(meta.DrawH) / float64(ih)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(scaleX*g.zoom, scaleY*g.zoom)
			op.GeoM.Translate(float64(ent.X)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(ent.Y)*g.zoom+g.panY)
			if i == g.selectedEntity {
				op.ColorScale.Scale(1, 1, 0.8, 1)
			}
			screen.DrawImage(img, op)
		} else {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
			op.GeoM.Translate(float64(ent.X)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(ent.Y)*g.zoom+g.panY)
			marker := color.RGBA{R: 255, G: 220, B: 80, A: 220}
			if i == g.selectedEntity {
				marker = color.RGBA{R: 255, G: 180, B: 40, A: 255}
			}
			op.ColorScale.Scale(float32(marker.R)/255, float32(marker.G)/255, float32(marker.B)/255, float32(marker.A)/255)
			screen.DrawImage(g.gridPixel, op)
		}
	}
	// Draw grid (limited to drawing canvas)
	rows := 0
	if len(g.layers) > 0 {
		rows = len(g.layers[0].Tiles)
	}
	cols := 0
	if rows > 0 {
		cols = len(g.layers[0].Tiles[0])
	}
	w := float64(cols * g.gridSize)
	h := float64(rows * g.gridSize)
	gridColor := color.RGBA{A: 64, R: 200, G: 200, B: 200}
	for x := 0; x <= cols; x++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(1, h*g.zoom)
		op.GeoM.Translate(float64(x*g.gridSize)*g.zoom+g.panX+float64(g.leftPanelWidth), g.panY)
		op.ColorScale.Scale(float32(gridColor.R)/255, float32(gridColor.G)/255, float32(gridColor.B)/255, float32(gridColor.A)/255)
		screen.DrawImage(g.gridPixel, op)
	}
	for y := 0; y <= rows; y++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(w*g.zoom, 1)
		op.GeoM.Translate(g.panX+float64(g.leftPanelWidth), float64(y*g.gridSize)*g.zoom+g.panY)
		op.ColorScale.Scale(float32(gridColor.R)/255, float32(gridColor.G)/255, float32(gridColor.B)/255, float32(gridColor.A)/255)
		screen.DrawImage(g.gridPixel, op)
	}
	// Draw selected tile preview under cursor (snapped to grid)
	previewDrawn := false
	if g.selectedPrefabName == "" && !g.transitionMode {
		if g.selectedTileset != nil && g.selectedTileIndex >= 0 {
			tileSize := g.gridSize
			tsW, tsH := g.selectedTileset.Size()
			tilesX := tsW / tileSize
			if tilesX > 0 {
				tileX := g.selectedTileIndex % tilesX
				tileY := g.selectedTileIndex / tilesX
				if tileX*tileSize < tsW && tileY*tileSize < tsH {
					sub := g.selectedTileset.SubImage(
						image.Rect(tileX*tileSize, tileY*tileSize, (tileX+1)*tileSize, (tileY+1)*tileSize),
					).(*ebiten.Image)
					cx, cy := ebiten.CursorPosition()
					screenW, _ := ebiten.Monitor().Size()
					if cx >= g.leftPanelWidth && cy >= 0 && cx < screenW+g.rightPanelWidth {
						worldX := (float64(cx-g.leftPanelWidth) - g.panX) / g.zoom
						worldY := (float64(cy) - g.panY) / g.zoom
						cellX := (int(worldX) / g.gridSize) * g.gridSize
						cellY := (int(worldY) / g.gridSize) * g.gridSize
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Scale(g.zoom, g.zoom)
						op.GeoM.Translate(float64(cellX)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(cellY)*g.zoom+g.panY)
						op.ColorScale.Scale(1, 1, 1, 0.5)
						screen.DrawImage(sub, op)
						previewDrawn = true
					}
				}
			}
		}
	}
	_ = previewDrawn
	// Draw prefab placement preview
	if g.selectedPrefabName != "" && !g.transitionMode {
		cx, cy := ebiten.CursorPosition()
		screenW, _ := ebiten.Monitor().Size()
		if cx >= g.leftPanelWidth && cy >= 0 && cx < screenW+g.rightPanelWidth {
			worldX := (float64(cx-g.leftPanelWidth) - g.panX) / g.zoom
			worldY := (float64(cy) - g.panY) / g.zoom
			cellX := (int(worldX) / g.gridSize) * g.gridSize
			cellY := (int(worldY) / g.gridSize) * g.gridSize
			if g.selectedPrefabImage != nil {
				img := g.selectedPrefabImage
				w, _ := img.Size()
				scale := float64(g.gridSize) / float64(w)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(scale*g.zoom, scale*g.zoom)
				op.GeoM.Translate(float64(cellX)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(cellY)*g.zoom+g.panY)
				op.ColorScale.Scale(1, 1, 1, 0.6)
				screen.DrawImage(img, op)
			} else {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
				op.GeoM.Translate(float64(cellX)*g.zoom+g.panX+float64(g.leftPanelWidth), float64(cellY)*g.zoom+g.panY)
				previewColor := color.RGBA{R: 255, G: 220, B: 80, A: 200}
				op.ColorScale.Scale(float32(previewColor.R)/255, float32(previewColor.G)/255, float32(previewColor.B)/255, float32(previewColor.A)/255)
				screen.DrawImage(g.gridPixel, op)
			}
		}
	}
	// Draw UI
	if g.ui != nil {
		g.ui.Draw(screen)
	}
}

func (g *EditorGame) normalizeSavePath(name string) string {
	base := filepath.Base(name)
	if filepath.Ext(base) == "" {
		base += ".json"
	}
	return filepath.Join("levels", base)
}

func (g *EditorGame) SaveLevelToPath(path string) error {
	if path == "" {
		return fmt.Errorf("empty save path")
	}
	level := levels.Level{
		Width:        g.gridCols,
		Height:       g.gridRows,
		Layers:       make([][]int, len(g.layers)),
		TilesetUsage: make([][]*levels.TileInfo, len(g.layers)),
		LayerMeta:    make([]levels.LayerMeta, len(g.layers)),
		Entities:     cloneEntities(g.entities),
	}
	for li, layer := range g.layers {
		flat := make([]int, g.gridCols*g.gridRows)
		usage := make([]*levels.TileInfo, g.gridCols*g.gridRows)
		for y := 0; y < g.gridRows; y++ {
			for x := 0; x < g.gridCols; x++ {
				idx := y*g.gridCols + x
				if y < len(layer.Tiles) && x < len(layer.Tiles[y]) {
					flat[idx] = layer.Tiles[y][x]
				}
				if y < len(layer.TilesetUsage) && x < len(layer.TilesetUsage[y]) {
					if layer.TilesetUsage[y][x] != nil {
						copyInfo := *layer.TilesetUsage[y][x]
						usage[idx] = &copyInfo
					}
				}
			}
		}
		level.Layers[li] = flat
		level.TilesetUsage[li] = usage
		level.LayerMeta[li] = levels.LayerMeta{Physics: layer.Physics}
	}

	data, err := json.MarshalIndent(level, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	g.savePath = path
	log.Printf("Saved level: %s", path)
	return nil
}

func (g *EditorGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Use the monitor size for fullscreen
	return ebiten.Monitor().Size()
}

func main() {
	assetsDir := flag.String("dir", "assets", "Directory containing tileset images")
	levelName := flag.String("level", "", "Level name to load from levels/ (basename or filename, .json optional)")
	autoMapPath := flag.String("autotile-map", "", "Optional JSON file with 47-tile indices in mask order")
	flag.Parse()

	log.Println("Editor starting...")
	if *autoMapPath != "" {
		data, err := os.ReadFile(*autoMapPath)
		if err != nil {
			log.Printf("Failed to read autotile map: %v", err)
		} else {
			var indices []int
			if err := json.Unmarshal(data, &indices); err != nil {
				log.Printf("Failed to parse autotile map: %v", err)
			} else if len(indices) != len(auto47MaskOrder) {
				log.Printf("Autotile map must have %d entries, got %d", len(auto47MaskOrder), len(indices))
			} else {
				auto47TileIndices = indices
				log.Printf("Loaded autotile map with %d entries", len(auto47TileIndices))
			}
		}
	}
	assets, err := ListImageAssets(*assetsDir)
	if err != nil {
		log.Fatalf("Failed to list assets: %v", err)
	}
	prefabs, err := ListPrefabs("prefabs")
	if err != nil {
		log.Printf("Failed to list prefabs: %v", err)
	}

	ebiten.SetFullscreen(true)

	var selectedTileset *ebiten.Image
	var tilesetZoom *TilesetGridZoomable

	gridSize := 32
	leftPanelWidth := 200
	panelWidth := 240
	w, h := ebiten.Monitor().Size()
	gridWidth := w - panelWidth - leftPanelWidth
	if gridWidth < gridSize {
		gridWidth = gridSize
	}
	cols := gridWidth / gridSize
	rows := h / gridSize
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	// If a -level flag was provided, try to load that level from the embedded levels/ FS
	var loadedLayers []DummyLayer
	var loadedSavePath string
	var loadedTilesetRelPath string
	var loadedEntities []levels.Entity
	if *levelName != "" {
		name := *levelName
		if filepath.Ext(name) == "" {
			name += ".json"
		}
		lvl, err := levels.LoadLevelFromFS(name)
		if err != nil {
			log.Printf("Failed to load level %s: %v", name, err)
		} else {
			// Use level dimensions
			cols = lvl.Width
			rows = lvl.Height
			// Recompute gridWidth to match columns
			gridWidth = cols * gridSize

			// Convert each flat layer into a DummyLayer 2D tiles grid
			for li := range lvl.Layers {
				flat := lvl.Layers[li]
				tiles := make([][]int, rows)
				for y := 0; y < rows; y++ {
					tiles[y] = make([]int, cols)
					for x := 0; x < cols; x++ {
						idx := y*cols + x
						if idx < len(flat) {
							tiles[y][x] = flat[idx]
						}
					}
				}
				var usage [][]*levels.TileInfo
				if li < len(lvl.TilesetUsage) && lvl.TilesetUsage[li] != nil {
					flatUsage := lvl.TilesetUsage[li]
					usage = make([][]*levels.TileInfo, rows)
					for y := 0; y < rows; y++ {
						usage[y] = make([]*levels.TileInfo, cols)
						for x := 0; x < cols; x++ {
							idx := y*cols + x
							if idx < len(flatUsage) {
								if flatUsage[idx] != nil {
									copyInfo := *flatUsage[idx]
									usage[y][x] = &copyInfo
									if loadedTilesetRelPath == "" && copyInfo.Path != "" {
										loadedTilesetRelPath = copyInfo.Path
									}
								}
							}
						}
					}
				} else {
					usage = make([][]*levels.TileInfo, rows)
					for y := range usage {
						usage[y] = make([]*levels.TileInfo, cols)
					}
				}
				name := fmt.Sprintf("Layer %d", li)
				if li == 0 {
					name = "Background"
				} else if li == 1 {
					name = "Physics"
				}
				loadedLayers = append(loadedLayers, DummyLayer{
					Name:         name,
					Tiles:        tiles,
					TilesetUsage: usage,
					Visible:      true,
					Tint:         color.RGBA{R: 100, G: 200, B: 255, A: 255},
					Physics:      (li == 1),
				})
			}
			loadedSavePath = name
			if len(lvl.Entities) > 0 {
				loadedEntities = cloneEntities(lvl.Entities)
			}
		}
	}
	if len(loadedLayers) == 0 {
		cols, rows = promptLevelDimensions(cols, rows)
		if cols < 1 {
			cols = 1
		}
		if rows < 1 {
			rows = 1
		}
		gridWidth = cols * gridSize
	}
	// Create an empty layer sized to the screen grid
	newTiles := func() [][]int {
		tiles := make([][]int, rows)
		for y := range tiles {
			tiles[y] = make([]int, cols)
		}
		return tiles
	}
	newTilesetUsage := func() [][]*levels.TileInfo {
		usage := make([][]*levels.TileInfo, rows)
		for y := range usage {
			usage[y] = make([]*levels.TileInfo, cols)
		}
		return usage
	}
	defaultLayer := DummyLayer{
		Name:         "Background",
		Tiles:        newTiles(),
		TilesetUsage: newTilesetUsage(),
		Visible:      true,
		Tint:         color.RGBA{R: 100, G: 200, B: 255, A: 255},
		Physics:      false,
	}
	secondLayer := DummyLayer{
		Name:         "Physics",
		Tiles:        newTiles(),
		TilesetUsage: newTilesetUsage(),
		Visible:      true,
		Tint:         color.RGBA{R: 100, G: 200, B: 255, A: 255},
		Physics:      true,
	}

	layersToUse := []DummyLayer{defaultLayer, secondLayer}
	if len(loadedLayers) > 0 {
		layersToUse = loadedLayers
	}

	game := &EditorGame{
		gridSize:          gridSize,
		gridWidth:         gridWidth,
		layers:            layersToUse,
		currentLayer:      0,
		tilesetZoom:       tilesetZoom,
		currentTool:       ToolBrush,
		lastTool:          ToolBrush,
		selectedTileIndex: -1,
		selectedEntity:    -1,
		zoom:              1.0,
		panX:              0,
		panY:              0,
		leftPanelWidth:    leftPanelWidth,
		rightPanelWidth:   panelWidth,
		gridRows:          rows,
		gridCols:          cols,
		maxUndo:           100,
		assetsDir:         *assetsDir,
		savePath:          loadedSavePath,
		entities:          loadedEntities,
		autotileEnabled:   true,
	}

	var (
		ui                         *ebitenui.UI
		toolBar                    *ToolBar
		layerPanel                 *LayerPanel
		fileNameInput              *widget.TextInput
		applyTileset               func(img *ebiten.Image)
		setTilesetSelection        func(tileIndex int)
		setTilesetSelectionEnabled func(enabled bool)
		transitionUI               *TransitionUI
	)

	ui, toolBar, layerPanel, fileNameInput, applyTileset, setTilesetSelection, setTilesetSelectionEnabled, transitionUI = BuildEditorUI(assets, prefabs, func(asset AssetInfo, setTileset func(img *ebiten.Image)) {
		f, err := os.Open(asset.Path)
		if err != nil {
			log.Printf("Failed to open asset: %v", err)
			return
		}
		defer f.Close()
		img, err := png.Decode(f)
		if err != nil {
			log.Printf("Failed to decode PNG: %v", err)
			return
		}
		selectedTileset = ebiten.NewImageFromImage(img)
		game.selectedTileset = selectedTileset
		relPath, relErr := filepath.Rel(*assetsDir, asset.Path)
		if relErr != nil {
			relPath = asset.Name
		}
		game.selectedTilesetPath = filepath.ToSlash(relPath)
		setTileset(selectedTileset)
		if setTilesetSelectionEnabled != nil {
			setTilesetSelectionEnabled(!game.autotileEnabled)
		}
		log.Printf("Tileset loaded: %s", asset.Name)
	}, func(tool Tool) {
		game.currentTool = tool
	}, func(tileIndex int) {
		game.selectedTileIndex = tileIndex
	}, func(layerIndex int) {
		game.currentLayer = layerIndex
		game.updatePhysicsButtonLabel()
	}, func(layerIndex int, newName string) {
		if layerIndex >= 0 && layerIndex < len(game.layers) {
			game.pushUndo()
			game.layers[layerIndex].Name = newName
			if game.layerPanel != nil {
				game.layerPanel.SetLayers(game.layerNames())
				game.layerPanel.SetSelected(game.currentLayer)
			}
		}
	}, func() {
		game.AddLayer()
	}, func(layerIndex int) {
		game.MoveLayerUp(layerIndex)
	}, func(layerIndex int) {
		game.MoveLayerDown(layerIndex)
	}, func() {
		game.TogglePhysicsForCurrentLayer()
	}, func() {
		game.showPhysicsHighlight = !game.showPhysicsHighlight
	}, func() {
		game.ToggleAutotile()
	}, func(prefab PrefabInfo) {
		game.transitionMode = false
		game.selectedPrefabName = prefab.Name
		game.selectedPrefabPath = prefab.Path
		game.selectedEntity = -1
		game.entityDragging = false
		game.syncTransitionUI()
		// Clear previous preview image
		game.selectedPrefabImage = nil
		// Try to load prefab spec bytes and inspect for animation/sprite
		data, err := prefabsPkg.Load(prefab.Path)
		if err != nil {
			log.Printf("prefab load failed: %v", err)
			return
		}
		var spec struct {
			Animation *prefabsPkg.AnimationSpec `yaml:"animation"`
			Sprite    *prefabsPkg.SpriteSpec    `yaml:"sprite"`
		}
		if err := yaml.Unmarshal(data, &spec); err != nil {
			log.Printf("prefab unmarshal failed: %v", err)
			return
		}
		// Rule 1: animation first def first frame
		if spec.Animation != nil && len(spec.Animation.Defs) > 0 {
			// pick first def deterministically by sorted key
			keys := make([]string, 0, len(spec.Animation.Defs))
			for k := range spec.Animation.Defs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			def := spec.Animation.Defs[keys[0]]
			// load sheet image from embedded assets
			sheet, err := assetsPkg.LoadImage(spec.Animation.Sheet)
			if err == nil && sheet != nil {
				x := def.ColStart * def.FrameW
				y := def.Row * def.FrameH
				r := image.Rect(x, y, x+def.FrameW, y+def.FrameH)
				if sub := sheet.SubImage(r); sub != nil {
					if bi, ok := sub.(*ebiten.Image); ok {
						game.selectedPrefabImage = bi
						game.selectedPrefabDrawW = def.FrameW
						game.selectedPrefabDrawH = def.FrameH
						if game.prefabMetaCache == nil {
							game.prefabMetaCache = make(map[string]prefabImageMeta)
						}
						game.prefabMetaCache[prefab.Path] = prefabImageMeta{Img: bi, DrawW: def.FrameW, DrawH: def.FrameH}
					}
				}
			}
		}
		// Rule 2: sprite image
		if game.selectedPrefabImage == nil && spec.Sprite != nil && spec.Sprite.Image != "" {
			if img, err := assetsPkg.LoadImage(spec.Sprite.Image); err == nil {
				game.selectedPrefabImage = img
				w, h := img.Size()
				game.selectedPrefabDrawW = w
				game.selectedPrefabDrawH = h
				if game.prefabMetaCache == nil {
					game.prefabMetaCache = make(map[string]prefabImageMeta)
				}
				game.prefabMetaCache[prefab.Path] = prefabImageMeta{Img: img, DrawW: w, DrawH: h}
			}
		}
		// If still nil, we'll draw the fallback square in Draw()
	}, func(enabled bool) {
		game.transitionMode = enabled
		if enabled {
			game.selectedPrefabName = ""
			game.selectedPrefabPath = ""
			game.entityDragging = false
			game.entityPendingUndo = false
		}
		game.syncTransitionUI()
	}, func(field string, value string) {
		if game.selectedEntity < 0 || game.selectedEntity >= len(game.entities) {
			return
		}
		if !game.isTransitionEntity(game.selectedEntity) {
			return
		}
		if game.transitionPropUndo {
			game.pushUndo()
			game.transitionPropUndo = false
		}
		ent := game.entities[game.selectedEntity]
		if ent.Props == nil {
			ent.Props = map[string]interface{}{}
		}
		ent.Props[field] = value
		game.entities[game.selectedEntity] = ent
	}, game.layerNames(), game.currentLayer, game.currentTool, game.autotileEnabled)

	game.ui = ui
	game.toolBar = toolBar
	game.layerPanel = layerPanel
	game.fileNameInput = fileNameInput
	game.setTilesetSelect = setTilesetSelection
	game.setTilesetSelectEnabled = setTilesetSelectionEnabled
	game.transitionUI = transitionUI
	game.syncTransitionUI()
	if game.savePath != "" && game.fileNameInput != nil {
		game.fileNameInput.SetText(game.savePath)
	}

	// If the loaded level referenced a tileset, try to find it in the assets
	if loadedTilesetRelPath != "" && applyTileset != nil {
		// Find matching asset by relative path under assetsDir
		found := false
		for _, a := range assets {
			relp, relErr := filepath.Rel(*assetsDir, a.Path)
			if relErr != nil {
				continue
			}
			if filepath.ToSlash(relp) == loadedTilesetRelPath {
				// load image and apply
				f, err := os.Open(a.Path)
				if err != nil {
					log.Printf("Failed to open tileset asset %s: %v", a.Path, err)
					break
				}
				img, err := png.Decode(f)
				f.Close()
				if err != nil {
					log.Printf("Failed to decode tileset PNG %s: %v", a.Path, err)
					break
				}
				selImg := ebiten.NewImageFromImage(img)
				game.selectedTileset = selImg
				game.selectedTilesetPath = filepath.ToSlash(relp)
				applyTileset(selImg)
				found = true
				break
			}
		}
		if !found {
			log.Printf("Referenced tileset not found in assets/: %s", loadedTilesetRelPath)
		}
	}

	game.updatePhysicsButtonLabel()
	game.updateAutotileButtonLabel()
	if game.setTilesetSelectEnabled != nil {
		game.setTilesetSelectEnabled(!game.autotileEnabled)
	}

	ebiten.SetWindowTitle("Tileset Editor")

	// Tileset zoom and panning logic should be handled in Update, not here

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// bresenhamLine returns a slice of [2]int points from (x0, y0) to (x1, y1)
func bresenhamLine(x0, y0, x1, y1 int) [][2]int {
	var points [][2]int
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy
	for {
		points = append(points, [2]int{x0, y0})
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
	return points
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
