package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// DummyLayer is a simple layer for demonstration.
type DummyLayer struct {
	Tiles   [][]int
	Visible bool
	Tint    color.RGBA
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

type EditorGame struct {
	lineStart         *[2]int // nil if not started
	ui                *ebitenui.UI
	gridSize          int
	gridWidth         int
	layer             DummyLayer
	tilesetZoom       *TilesetGridZoomable
	currentTool       Tool
	lastTool          Tool
	toolBar           *ToolBar
	selectedTileset   *ebiten.Image
	selectedTileIndex int
	zoom              float64
	panX              float64
	panY              float64
	isPanning         bool
	lastPanX          int
	lastPanY          int
	gridPixel         *ebiten.Image
}

// floodFill fills contiguous tiles of the same value starting from (x, y)
func (g *EditorGame) floodFill(x, y, target, replacement int) {
	if target == replacement {
		return
	}
	if y < 0 || y >= len(g.layer.Tiles) || x < 0 || x >= len(g.layer.Tiles[y]) {
		return
	}
	if g.layer.Tiles[y][x] != target {
		return
	}
	g.layer.Tiles[y][x] = replacement
	g.floodFill(x+1, y, target, replacement)
	g.floodFill(x-1, y, target, replacement)
	g.floodFill(x, y+1, target, replacement)
	g.floodFill(x, y-1, target, replacement)
}

func (g *EditorGame) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		os.Exit(0)
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
	if sx < 0 || sy < 0 || sx >= g.gridWidth {
		return nil
	}
	worldX := (float64(sx) - g.panX) / g.zoom
	worldY := (float64(sy) - g.panY) / g.zoom
	if worldX < 0 || worldY < 0 {
		return nil
	}
	cellX := int(worldX) / g.gridSize
	cellY := int(worldY) / g.gridSize
	// Brush/Erase/Fill/Line tool logic
	if cellY >= 0 && cellY < len(g.layer.Tiles) && cellX >= 0 && cellX < len(g.layer.Tiles[cellY]) {
		switch g.currentTool {
		case ToolBrush:
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				if g.selectedTileIndex >= 0 {
					g.layer.Tiles[cellY][cellX] = g.selectedTileIndex + 1
				}
			}
		case ToolErase:
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				g.layer.Tiles[cellY][cellX] = 0 // Erase tile
			}
		case ToolFill:
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				if g.selectedTileIndex >= 0 {
					start := g.layer.Tiles[cellY][cellX]
					replace := g.selectedTileIndex + 1
					g.floodFill(cellX, cellY, start, replace)
				}
			}
		case ToolLine:
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				// Set start point
				g.lineStart = &[2]int{cellX, cellY}
			}
			if g.lineStart != nil && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
				// Set end point and draw line
				x0, y0 := g.lineStart[0], g.lineStart[1]
				x1, y1 := cellX, cellY
				for _, pt := range bresenhamLine(x0, y0, x1, y1) {
					px, py := pt[0], pt[1]
					if py >= 0 && py < len(g.layer.Tiles) && px >= 0 && px < len(g.layer.Tiles[py]) {
						if g.selectedTileIndex >= 0 {
							g.layer.Tiles[py][px] = g.selectedTileIndex + 1
						}
					}
				}
				g.lineStart = nil
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
	// Draw tiled layer (if visible)
	if g.layer.Visible {
		for y, row := range g.layer.Tiles {
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
							op.GeoM.Translate(float64(x*g.gridSize)*g.zoom+g.panX, float64(y*g.gridSize)*g.zoom+g.panY)
							screen.DrawImage(sub, op)
							continue
						}
					}
				}
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
				op.GeoM.Translate(float64(x*g.gridSize)*g.zoom+g.panX, float64(y*g.gridSize)*g.zoom+g.panY)
				op.ColorScale.Scale(float32(g.layer.Tint.R)/255, float32(g.layer.Tint.G)/255, float32(g.layer.Tint.B)/255, 0.5)
				screen.DrawImage(g.gridPixel, op)
			}
		}
	}
	// Draw line preview
	if g.currentTool == ToolLine && g.lineStart != nil {
		cx, cy := ebiten.CursorPosition()
		if cx >= 0 && cy >= 0 && cx < g.gridWidth {
			worldX := (float64(cx) - g.panX) / g.zoom
			worldY := (float64(cy) - g.panY) / g.zoom
			endX := int(worldX) / g.gridSize
			endY := int(worldY) / g.gridSize
			startX, startY := g.lineStart[0], g.lineStart[1]
			for _, pt := range bresenhamLine(startX, startY, endX, endY) {
				px, py := pt[0], pt[1]
				if py < 0 || py >= len(g.layer.Tiles) || px < 0 || px >= len(g.layer.Tiles[py]) {
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
							op.GeoM.Translate(float64(px*g.gridSize)*g.zoom+g.panX, float64(py*g.gridSize)*g.zoom+g.panY)
							op.ColorScale.Scale(1, 1, 1, 0.5)
							screen.DrawImage(sub, op)
							continue
						}
					}
				}
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(float64(g.gridSize)*g.zoom, float64(g.gridSize)*g.zoom)
				op.GeoM.Translate(float64(px*g.gridSize)*g.zoom+g.panX, float64(py*g.gridSize)*g.zoom+g.panY)
				op.ColorScale.Scale(float32(g.layer.Tint.R)/255, float32(g.layer.Tint.G)/255, float32(g.layer.Tint.B)/255, 0.5)
				screen.DrawImage(g.gridPixel, op)
			}
		}
	}
	// Draw grid (limited to drawing canvas)
	rows := len(g.layer.Tiles)
	cols := 0
	if rows > 0 {
		cols = len(g.layer.Tiles[0])
	}
	w := float64(cols * g.gridSize)
	h := float64(rows * g.gridSize)
	gridColor := color.RGBA{A: 64, R: 200, G: 200, B: 200}
	for x := 0; x <= cols; x++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(1, h*g.zoom)
		op.GeoM.Translate(float64(x*g.gridSize)*g.zoom+g.panX, g.panY)
		op.ColorScale.Scale(float32(gridColor.R)/255, float32(gridColor.G)/255, float32(gridColor.B)/255, float32(gridColor.A)/255)
		screen.DrawImage(g.gridPixel, op)
	}
	for y := 0; y <= rows; y++ {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(w*g.zoom, 1)
		op.GeoM.Translate(g.panX, float64(y*g.gridSize)*g.zoom+g.panY)
		op.ColorScale.Scale(float32(gridColor.R)/255, float32(gridColor.G)/255, float32(gridColor.B)/255, float32(gridColor.A)/255)
		screen.DrawImage(g.gridPixel, op)
	}
	// Draw selected tile preview under cursor (snapped to grid)
	previewDrawn := false
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
				if cx >= 0 && cy >= 0 && cx < g.gridWidth {
					worldX := (float64(cx) - g.panX) / g.zoom
					worldY := (float64(cy) - g.panY) / g.zoom
					cellX := (int(worldX) / g.gridSize) * g.gridSize
					cellY := (int(worldY) / g.gridSize) * g.gridSize
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(g.zoom, g.zoom)
					op.GeoM.Translate(float64(cellX)*g.zoom+g.panX, float64(cellY)*g.zoom+g.panY)
					op.ColorScale.Scale(1, 1, 1, 0.5)
					screen.DrawImage(sub, op)
					previewDrawn = true
				}
			}
		}
	}
	_ = previewDrawn
	// Draw UI
	if g.ui != nil {
		g.ui.Draw(screen)
	}
}

func (g *EditorGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Use the monitor size for fullscreen
	return ebiten.Monitor().Size()
}

func main() {
	log.Println("Editor starting...")
	assets, err := ListImageAssets()
	if err != nil {
		log.Fatalf("Failed to list assets: %v", err)
	}

	ebiten.SetFullscreen(true)

	var selectedTileset *ebiten.Image
	var tilesetZoom *TilesetGridZoomable

	gridSize := 32
	panelWidth := 240
	w, h := ebiten.Monitor().Size()
	gridWidth := w - panelWidth
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
	// Create an empty layer sized to the screen grid
	tiles := make([][]int, rows)
	for y := range tiles {
		tiles[y] = make([]int, cols)
	}
	layer := DummyLayer{
		Tiles:   tiles,
		Visible: true,
		Tint:    color.RGBA{R: 100, G: 200, B: 255, A: 255},
	}

	game := &EditorGame{
		gridSize:          gridSize,
		gridWidth:         gridWidth,
		layer:             layer,
		tilesetZoom:       tilesetZoom,
		currentTool:       ToolBrush,
		lastTool:          ToolBrush,
		selectedTileIndex: -1,
		zoom:              1.0,
		panX:              0,
		panY:              0,
	}

	ui, toolBar := BuildEditorUI(assets, func(asset AssetInfo, setTileset func(img *ebiten.Image)) {
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
		setTileset(selectedTileset)
		log.Printf("Tileset loaded: %s", asset.Name)
	}, func(tool Tool) {
		game.currentTool = tool
	}, func(tileIndex int) {
		game.selectedTileIndex = tileIndex
	}, game.currentTool)

	game.ui = ui
	game.toolBar = toolBar

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
