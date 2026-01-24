package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Level struct {
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Layers    [][]int     `json:"layers,omitempty"` // optional layers, each row-major
	LayerMeta []LayerMeta `json:"layer_meta,omitempty"`
	SpawnX    int         `json:"spawn_x,omitempty"`
	SpawnY    int         `json:"spawn_y,omitempty"`
}

type LayerMeta struct {
	HasPhysics bool   `json:"has_physics"`
	Color      string `json:"color"`
}

type Editor struct {
	level             *Level
	cellSize          int
	backgroundTileImg *ebiten.Image
	tileImg           *ebiten.Image
	foregroundTileImg *ebiten.Image
	emptyImg          *ebiten.Image
	hoverImg          *ebiten.Image
	prevMouse         bool
	filename          string
	currentLayer      int
	prevCyclePrev     bool
	prevCycleNext     bool
	// per-layer rendered images (one per layer) matching colors in LayerMeta
	layerTileImgs []*ebiten.Image
	// spawn placement
	spawnMode     bool
	spawnImg      *ebiten.Image
	spawnImgHover *ebiten.Image
	// triangle placement (only on physics-enabled layers)
	triangleMode     bool
	triangleImg      *ebiten.Image
	triangleImgHover *ebiten.Image
}

const (
	baseWidthEditor  = 40 * 32 // 1280
	baseHeightEditor = 23 * 32 // 736
)

// NewEditor creates an EditorGame with cell size; call Init or Load before running.
func NewEditor(cellSize int) *Editor {
	eg := &Editor{cellSize: cellSize}

	eg.tileImg = ebiten.NewImage(cellSize, cellSize)
	eg.tileImg.Fill(color.RGBA{R: 0, G: 0, B: 0xff, A: 0xff})

	eg.emptyImg = ebiten.NewImage(cellSize, cellSize)
	eg.emptyImg.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0xff})

	eg.hoverImg = ebiten.NewImage(cellSize, cellSize)
	eg.hoverImg.Fill(color.RGBA{R: 128, G: 128, B: 128, A: 0x88})

	eg.spawnImg = circleImage(cellSize, color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0x88})
	eg.spawnImgHover = circleImage(cellSize, color.RGBA{R: 128, G: 128, B: 128, A: 0x88})

	eg.triangleImg = triangleImage(cellSize, color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})
	eg.triangleImgHover = triangleImage(cellSize, color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0x88})

	return eg
}

// Init initializes a new empty level with given width/height in cells.
func (g *Editor) Init(w, h int) {
	// start with only layer 0 by default
	layers := make([][]int, 1)
	layers[0] = make([]int, w*h)
	meta := make([]LayerMeta, 1)
	meta[0] = LayerMeta{HasPhysics: false, Color: "#3c78ff"}
	g.level = &Level{Width: w, Height: h, Layers: layers, LayerMeta: meta}
	g.currentLayer = 0
	// setup per-layer images
	g.layerTileImgs = make([]*ebiten.Image, len(g.level.LayerMeta))
	for i := range g.level.LayerMeta {
		g.layerTileImgs[i] = layerImageFromHex(g.cellSize, g.level.LayerMeta[i].Color)
	}
}

func (g *Editor) Update() error {
	// Mouse toggle on press (edge)
	mx, my := ebiten.CursorPosition()
	w := g.level.Width * g.cellSize
	h := g.level.Height * g.cellSize

	// Toggle spawn placement mode (P). While active, left-click places spawn.
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.spawnMode = !g.spawnMode
	}

	if g.spawnMode && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// set spawn to hovered cell
		if mx >= 0 && my >= 0 && mx < w && my < h {
			gx := mx / g.cellSize
			gy := my / g.cellSize
			g.level.SpawnX = gx
			g.level.SpawnY = gy
		}
	}

	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if pressed && !g.prevMouse && !g.spawnMode {
		if mx >= 0 && my >= 0 && mx < w && my < h {
			cx := mx / g.cellSize
			cy := my / g.cellSize
			idx := cy*g.level.Width + cx
			if idx >= 0 && idx < g.level.Width*g.level.Height {
				// ensure Layers exists
				if g.level.Layers == nil || len(g.level.Layers) == 0 {
					g.level.Layers = make([][]int, 1)
					g.level.Layers[0] = make([]int, g.level.Width*g.level.Height)
				}
				layer := g.level.Layers[g.currentLayer]
				// triangle placement mode (only allowed on physics-enabled layers)
				canTriangle := false
				if g.level.LayerMeta != nil && g.currentLayer < len(g.level.LayerMeta) {
					canTriangle = g.level.LayerMeta[g.currentLayer].HasPhysics
				}
				if g.triangleMode && canTriangle {
					if layer[idx] == 2 {
						layer[idx] = 0
					} else {
						layer[idx] = 2
					}
				} else {
					// normal tile placement toggles between 0 and 1 (clears 2 as well)
					if layer[idx] == 0 {
						layer[idx] = 1
					} else {
						layer[idx] = 0
					}
				}
				g.level.Layers[g.currentLayer] = layer
			}
		}
	}
	g.prevMouse = pressed

	// Cycle layers: Q = previous, E = next (edge-detected)
	cyclePrev := ebiten.IsKeyPressed(ebiten.KeyQ)
	if cyclePrev && !g.prevCyclePrev {
		if g.level.Layers == nil || len(g.level.Layers) == 0 {
			g.currentLayer = 0
		} else {
			g.currentLayer--
			if g.currentLayer < 0 {
				g.currentLayer = len(g.level.Layers) - 1
			}
		}
	}
	g.prevCyclePrev = cyclePrev

	cycleNext := ebiten.IsKeyPressed(ebiten.KeyE)
	if cycleNext && !g.prevCycleNext {
		if g.level.Layers == nil || len(g.level.Layers) == 0 {
			g.currentLayer = 0
		} else {
			g.currentLayer++
			if g.currentLayer >= len(g.level.Layers) {
				g.currentLayer = 0
			}
		}
	}
	g.prevCycleNext = cycleNext

	// Create a new layer (N)
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		newLayer := make([]int, g.level.Width*g.level.Height)
		g.level.Layers = append(g.level.Layers, newLayer)
		// default meta for new layer
		g.level.LayerMeta = append(g.level.LayerMeta, LayerMeta{HasPhysics: false, Color: "#3c78ff"})
		// create image for new layer
		g.layerTileImgs = append(g.layerTileImgs, layerImageFromHex(g.cellSize, "#3c78ff"))
		g.currentLayer = len(g.level.Layers) - 1
	}

	// Toggle physics flag for current layer (H)
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		if g.level.LayerMeta == nil || g.currentLayer >= len(g.level.LayerMeta) {
			// ensure meta exists
			for len(g.level.LayerMeta) <= g.currentLayer {
				g.level.LayerMeta = append(g.level.LayerMeta, LayerMeta{HasPhysics: false, Color: "#3c78ff"})
				g.layerTileImgs = append(g.layerTileImgs, layerImageFromHex(g.cellSize, "#3c78ff"))
			}
		}
		g.level.LayerMeta[g.currentLayer].HasPhysics = !g.level.LayerMeta[g.currentLayer].HasPhysics
	}

	// Toggle triangle mode (T) — only enabled if current layer has physics
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		if g.level != nil && g.level.LayerMeta != nil && g.currentLayer < len(g.level.LayerMeta) && g.level.LayerMeta[g.currentLayer].HasPhysics {
			g.triangleMode = !g.triangleMode
		}
	}

	// Cycle color for current layer (C)
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		palette := []string{"#3c78ff", "#40c040", "#ff8040", "#c040c0", "#ffffff"}
		if g.level.LayerMeta == nil || g.currentLayer >= len(g.level.LayerMeta) {
			// ensure meta exists
			for len(g.level.LayerMeta) <= g.currentLayer {
				g.level.LayerMeta = append(g.level.LayerMeta, LayerMeta{HasPhysics: false, Color: palette[0]})
				g.layerTileImgs = append(g.layerTileImgs, layerImageFromHex(g.cellSize, palette[0]))
			}
		}
		cur := g.level.LayerMeta[g.currentLayer].Color
		// find in palette
		idx := 0
		for i, c := range palette {
			if c == cur {
				idx = i
				break
			}
		}
		next := palette[(idx+1)%len(palette)]
		g.level.LayerMeta[g.currentLayer].Color = next
		// update image
		if g.currentLayer < len(g.layerTileImgs) {
			g.layerTileImgs[g.currentLayer] = layerImageFromHex(g.cellSize, next)
		}
	}
	// Save if S pressed
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if err := g.Save(); err != nil {
			log.Printf("save error: %v", err)
		} else {
			log.Printf("saved to %s", g.filename)
		}
	}

	return nil
}

func (g *Editor) Draw(screen *ebiten.Image) {
	// Draw base empty grid once
	for y := 0; y < g.level.Height; y++ {
		for x := 0; x < g.level.Width; x++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
			screen.DrawImage(g.emptyImg, op)
		}
	}

	// Draw layers bottom-to-top (only draw tiles; background already drawn)
	for layerIdx := 0; layerIdx < len(g.level.Layers); layerIdx++ {
		tileImg := g.layerTileImgs[layerIdx]

		layer := g.level.Layers[layerIdx]
		for y := 0; y < g.level.Height; y++ {
			for x := 0; x < g.level.Width; x++ {
				idx := y*g.level.Width + x
				if layer[idx] == 1 {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
					screen.DrawImage(tileImg, op)
				} else if layer[idx] == 2 {
					// triangle marker
					if g.triangleImg != nil {
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
						screen.DrawImage(g.triangleImg, op)
					}
				}
			}
		}
	}

	// Hover highlight (draw on top)
	mx, my := ebiten.CursorPosition()
	if g.level != nil {
		if mx >= 0 && my >= 0 {
			gx := mx / g.cellSize
			gy := my / g.cellSize
			if gx >= 0 && gy >= 0 && gx < g.level.Width && gy < g.level.Height {
				hop := &ebiten.DrawImageOptions{}
				hop.GeoM.Translate(float64(gx*g.cellSize), float64(gy*g.cellSize))
				if g.spawnMode {
					screen.DrawImage(g.spawnImgHover, hop)
				} else if g.triangleMode {
					screen.DrawImage(g.triangleImgHover, hop)
				} else {
					screen.DrawImage(g.hoverImg, hop)
				}
			}
		}
	}

	// Draw spawn marker: if spawnMode active show at hover cell, else at saved spawn
	if g.spawnImg != nil && g.level != nil {
		sx := g.level.SpawnX
		sy := g.level.SpawnY
		if sx >= 0 && sy >= 0 && sx < g.level.Width && sy < g.level.Height {
			sop := &ebiten.DrawImageOptions{}
			sop.GeoM.Translate(float64(sx*g.cellSize), float64(sy*g.cellSize))
			screen.DrawImage(g.spawnImg, sop)
		}
	}

	// Instructions
	// Show layer meta info and controls
	curMeta := LayerMeta{}
	if g.level != nil && g.level.LayerMeta != nil && g.currentLayer < len(g.level.LayerMeta) {
		curMeta = g.level.LayerMeta[g.currentLayer]
	}
	instr := fmt.Sprintf("Left-click: toggle tile   S: save   Q/E: cycle layers   N: new layer   H: toggle physics   C: cycle color   P: place spawn   T: triangle mode   File: %s\nW=%d H=%d Cell=%d Layer=%d has_physics=%v color=%s spawn=(%d,%d) spawnMode=%v triangleMode=%v",
		g.filename, g.level.Width, g.level.Height, g.cellSize, g.currentLayer, curMeta.HasPhysics, curMeta.Color, g.level.SpawnX, g.level.SpawnY, g.spawnMode, g.triangleMode)
	ebitenutil.DebugPrint(screen, instr)
}

func (g *Editor) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	// Fixed logical size that matches the grid: 40x23 cells at 32px each.
	return baseWidthEditor, baseHeightEditor
}

func (g *Editor) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("Layout called; use LayoutF instead")
}

func (g *Editor) Save() error {
	if g.filename == "" {
		// ensure levels dir
		if err := os.MkdirAll("levels", 0755); err != nil {
			return err
		}
		g.filename = filepath.Join("levels", fmt.Sprintf("level_%d.json", time.Now().Unix()))
	} else {
		// ensure directory exists
		if err := os.MkdirAll(filepath.Dir(g.filename), 0755); err != nil {
			return err
		}
	}
	f, err := os.Create(g.filename)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(g.level)
}

func (g *Editor) Load(filename string) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var lvl Level
	if err := json.Unmarshal(b, &lvl); err != nil {
		return err
	}

	// ensure there is at least one layer
	if lvl.Layers == nil || len(lvl.Layers) == 0 {
		lvl.Layers = make([][]int, 1)
		lvl.Layers[0] = make([]int, lvl.Width*lvl.Height)
	}

	// ensure layer meta exists for each layer
	if lvl.LayerMeta == nil || len(lvl.LayerMeta) < len(lvl.Layers) {
		// fill missing with defaults
		meta := make([]LayerMeta, len(lvl.Layers))
		for i := range meta {
			if lvl.LayerMeta != nil && i < len(lvl.LayerMeta) {
				meta[i] = lvl.LayerMeta[i]
			} else {
				meta[i] = LayerMeta{HasPhysics: false, Color: "#3c78ff"}
			}
		}
		lvl.LayerMeta = meta
	}

	g.level = &lvl
	if g.currentLayer >= len(g.level.Layers) {
		g.currentLayer = 0
	}
	// rebuild per-layer images
	g.layerTileImgs = make([]*ebiten.Image, len(g.level.LayerMeta))
	for i := range g.level.LayerMeta {
		g.layerTileImgs[i] = layerImageFromHex(g.cellSize, g.level.LayerMeta[i].Color)
	}

	g.filename = filename
	return nil
}

// layerImageFromHex creates an image filled with the provided hex color ("#rrggbb").
func layerImageFromHex(size int, hex string) *ebiten.Image {
	c := parseHexColor(hex)
	img := ebiten.NewImage(size, size)
	img.Fill(c)
	return img
}

// parseHexColor parses a color in the form #rrggbb. Returns opaque color if parse fails.
func parseHexColor(s string) color.RGBA {
	var r, g, b uint8 = 0x3c, 0x78, 0xff
	if len(s) == 7 && s[0] == '#' {
		var ri, gi, bi uint32
		if _, err := fmt.Sscanf(s[1:], "%02x%02x%02x", &ri, &gi, &bi); err == nil {
			r = uint8(ri)
			g = uint8(gi)
			b = uint8(bi)
		}
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xff}
}

// circleImage builds an RGBA image with a filled circle of the given color.
func circleImage(size int, col color.RGBA) *ebiten.Image {
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	cx := float64(size) / 2
	cy := float64(size) / 2
	r := float64(size)/2 - 2
	rr := r * r
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= rr {
				rgba.Set(x, y, col)
			} else {
				// transparent
				rgba.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return ebiten.NewImageFromImage(rgba)
}

// triangleImage builds an RGBA image with a filled upward-pointing triangle of the given color.
func triangleImage(size int, col color.RGBA) *ebiten.Image {
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	cx := float64(size) / 2
	// draw an upward triangle with base at bottom
	for y := 0; y < size; y++ {
		// row progress from top (0) to bottom (size-1)
		progress := float64(y) / float64(size-1)
		// width grows with progress
		rowWidth := progress * float64(size)
		left := cx - rowWidth/2
		right := cx + rowWidth/2
		for x := 0; x < size; x++ {
			fx := float64(x) + 0.5
			if fx >= left && fx <= right {
				rgba.Set(x, y, col)
			} else {
				rgba.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}
	return ebiten.NewImageFromImage(rgba)
}
