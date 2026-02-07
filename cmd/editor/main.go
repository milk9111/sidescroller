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
	lineStart   *[2]int // nil if not started
	ui          *ebitenui.UI
	gridSize    int
	layer       DummyLayer
	tilesetZoom *TilesetGridZoomable
	currentTool Tool
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

	if g.ui != nil {
		g.ui.Update()
	}
	// Mouse to grid mapping
	x, y := ebiten.CursorPosition()
	cellX := x / g.gridSize
	cellY := y / g.gridSize
	// Brush/Erase/Fill/Line tool logic
	if cellY >= 0 && cellY < len(g.layer.Tiles) && cellX >= 0 && cellX < len(g.layer.Tiles[cellY]) {
		switch g.currentTool {
		case ToolBrush:
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				g.layer.Tiles[cellY][cellX] = 1 // Paint with tile index 1 for now
			}
		case ToolErase:
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				g.layer.Tiles[cellY][cellX] = 0 // Erase tile
			}
		case ToolFill:
			if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
				start := g.layer.Tiles[cellY][cellX]
				g.floodFill(cellX, cellY, start, 1) // Fill with tile index 1 for now
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
						g.layer.Tiles[py][px] = 1 // Paint with tile index 1 for now
					}
				}
				g.lineStart = nil
			}
		}
	}
	return nil
}

func (g *EditorGame) Draw(screen *ebiten.Image) {
	// Draw tiled layer (if visible)
	if g.layer.Visible {
		for y, row := range g.layer.Tiles {
			for x, v := range row {
				if v == 0 {
					continue
				}
				rect := image.Rect(x*g.gridSize, y*g.gridSize, (x+1)*g.gridSize, (y+1)*g.gridSize)
				clr := g.layer.Tint
				clr.A = 128
				for py := rect.Min.Y; py < rect.Max.Y; py++ {
					for px := rect.Min.X; px < rect.Max.X; px++ {
						screen.Set(px, py, clr)
					}
				}
			}
		}
	}
	// Draw grid
	w, h := screen.Size()
	gridColor := color.RGBA{A: 64, R: 200, G: 200, B: 200}
	for x := 0; x < w; x += g.gridSize {
		for y := 0; y < h; y++ {
			screen.Set(x, y, gridColor)
		}
	}
	for y := 0; y < h; y += g.gridSize {
		for x := 0; x < w; x++ {
			screen.Set(x, y, gridColor)
		}
	}
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

	ui := BuildEditorUI(assets, func(asset AssetInfo, setTileset func(img *ebiten.Image)) {
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
		setTileset(selectedTileset)
		log.Printf("Tileset loaded: %s", asset.Name)
	})

	// Create a sample 10x8 layer with some tiles
	tiles := make([][]int, 8)
	for y := range tiles {
		tiles[y] = make([]int, 10)
		for x := range tiles[y] {
			if (x+y)%3 == 0 {
				tiles[y][x] = 1
			}
		}
	}
	layer := DummyLayer{
		Tiles:   tiles,
		Visible: true,
		Tint:    color.RGBA{R: 100, G: 200, B: 255, A: 255},
	}

	game := &EditorGame{
		ui:          ui,
		gridSize:    32,
		layer:       layer,
		tilesetZoom: tilesetZoom,
		currentTool: ToolBrush,
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
