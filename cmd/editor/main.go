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
type EditorGame struct {
	ui       *ebitenui.UI
	gridSize int
	layer    DummyLayer
}

func (g *EditorGame) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		os.Exit(0)
	}

	if g.ui != nil {
		g.ui.Update()
	}
	// Mouse to grid mapping
	x, y := ebiten.CursorPosition()
	cellX := x / g.gridSize
	cellY := y / g.gridSize
	// For demonstration, log the cell under the mouse
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		log.Printf("Mouse at cell: (%d, %d)", cellX, cellY)
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
	_ = selectedTileset // Prevent unused warning for now
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
		ui:       ui,
		gridSize: 32,
		layer:    layer,
	}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
