package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Editor struct {
	level *Level
	// background manager (entries + scaled images)
	backgrounds *Background
	// canvas manager
	canvas *Canvas
	// canvas transform for zoom/pan
	canvasOffsetX    float64
	canvasOffsetY    float64
	canvasZoom       float64
	canvasDragActive bool
	canvasLastMX     int
	canvasLastMY     int
	// left panel width (entities)
	cellSize          int
	tileImg           *ebiten.Image
	foregroundTileImg *ebiten.Image
	emptyImg          *ebiten.Image
	hoverImg          *ebiten.Image
	canvasImg         *ebiten.Image
	prevMouse         bool
	filename          string
	currentLayer      int
	prevCyclePrev     bool
	prevCycleNext     bool
	// drag paint state
	dragging      bool
	rightDragging bool
	paintValue    int
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

	highlightPhysics bool
	borderImg        *ebiten.Image
	// tileset panel component
	tilesetPanel *TilesetPanel
	entityPanel  *EntityPanel
	// controls text component
	controlsText ControlsText

	// missing image drawn when a tileset subimage can't be extracted
	missingImg *ebiten.Image
	// undo stack: stores past snapshots (full or delta) for undo
	undoStack []UndoSnapshot
	maxUndo   int
}

// NewEditor creates an EditorGame with cell size; call Init or Load before running.
func NewEditor(cellSize int, pprof bool) *Editor {
	eg := &Editor{cellSize: cellSize}

	eg.tileImg = ebiten.NewImage(cellSize, cellSize)
	eg.tileImg.Fill(color.RGBA{R: 0, G: 0, B: 0xff, A: 0xff})

	eg.emptyImg = ebiten.NewImage(cellSize, cellSize)
	eg.emptyImg.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0xff})

	eg.hoverImg = ebiten.NewImage(cellSize, cellSize)
	eg.hoverImg.Fill(color.RGBA{R: 128, G: 128, B: 128, A: 0x88})

	// missing / placeholder image (magenta)
	eg.missingImg = ebiten.NewImage(cellSize, cellSize)
	eg.missingImg.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff})

	eg.spawnImg = circleImage(cellSize, color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0x88})
	eg.spawnImgHover = circleImage(cellSize, color.RGBA{R: 128, G: 128, B: 128, A: 0x88})

	eg.triangleImg = triangleImage(cellSize, color.RGBA{R: 0xff, G: 0x00, B: 0x00, A: 0xff})
	eg.triangleImgHover = triangleImage(cellSize, color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0x88})

	// small 1px purple border used for physics highlighting
	bi := ebiten.NewImage(1, 1)
	bi.Fill(color.RGBA{R: 0x80, G: 0x00, B: 0x80, A: 0xff})
	eg.borderImg = bi

	eg.maxUndo = 5

	eg.tilesetPanel = NewTilesetPanel(
		184,
		220,
		cellSize,
		1.0,
	)

	eg.entityPanel = NewEntityPanel()

	// background manager
	eg.backgrounds = NewBackground()

	// canvas manager
	eg.canvas = NewCanvas()
	// ensure canvas knows left panel width
	eg.canvas.LeftPanelW = leftPanelWidth

	// canvas default transform
	eg.canvasZoom = 1.0
	eg.canvasOffsetX = 0
	eg.canvasOffsetY = 0

	// controls text defaults
	eg.controlsText = ControlsText{X: 8, Y: 8}

	if pprof {
		startPprofServer()
	}

	return eg
}

func init() {
	runtime.MemProfileRate = 1
}

func startPprofServer() {
	const addr = "127.0.0.1:6060"
	mux := http.NewServeMux()
	mux.Handle("/heap", pprof.Handler("heap"))
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	go func() {
		log.Printf("pprof server listening on http://%s", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()
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

// Update handles input and editor state changes.
func (g *Editor) Update() error {
	// Mouse toggle on press (edge)
	mx, my := ebiten.CursorPosition()
	// compute dynamic right-side panel X based on current window size
	winW, _ := ebiten.WindowSize()
	sideWidth := 220
	panelX := winW - sideWidth

	// Toggle spawn placement mode (P). While active, left-click places spawn.
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.spawnMode = !g.spawnMode
	}

	// helper: transform screen coords to canvas-local (unzoomed) coords and test inside canvas
	screenToCanvas := func(sx, sy int) (float64, float64, bool) {
		if sx < leftPanelWidth || sx >= panelX {
			return 0, 0, false
		}
		// local pixel inside canvas (relative to left panel)
		lx := float64(sx - leftPanelWidth)
		ly := float64(sy)
		// map through pan/zoom
		if g.canvasZoom == 0 {
			g.canvasZoom = 1.0
		}
		cx := (lx - g.canvasOffsetX) / g.canvasZoom
		cy := (ly - g.canvasOffsetY) / g.canvasZoom
		return cx, cy, true
	}

	if g.spawnMode && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if cx, cy, ok := screenToCanvas(mx, my); ok {
			gx := int(math.Floor(cx / float64(g.cellSize)))
			gy := int(math.Floor(cy / float64(g.cellSize)))
			if gx >= 0 && gy >= 0 && gx < g.level.Width && gy < g.level.Height {
				g.level.SpawnX = gx
				g.level.SpawnY = gy
			}
		}
	}

	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	inTilesetPanel := g.tilesetPanel.Update(mx, my, panelX, pressed, g.prevMouse)

	// Delegate canvas-interaction logic (sync Editor -> Canvas then update)
	if g.canvas != nil {
		// ensure canvas left panel width matches UI constant
		g.canvas.LeftPanelW = leftPanelWidth
		// sync editor state into canvas
		g.canvas.CanvasZoom = g.canvasZoom
		g.canvas.CanvasOffsetX = g.canvasOffsetX
		g.canvas.CanvasOffsetY = g.canvasOffsetY
		g.canvas.CanvasDragActive = g.canvasDragActive
		g.canvas.CanvasLastMX = g.canvasLastMX
		g.canvas.CanvasLastMY = g.canvasLastMY
		g.canvas.PrevMouse = g.prevMouse
		g.canvas.Dragging = g.dragging
		g.canvas.RightDragging = g.rightDragging
		g.canvas.PaintValue = g.paintValue
		g.canvas.Level = g.level
		g.canvas.CellSize = g.cellSize
		g.canvas.CurrentLayer = g.currentLayer
		g.canvas.EmptyImg = g.emptyImg
		g.canvas.HoverImg = g.hoverImg
		g.canvas.MissingImg = g.missingImg
		g.canvas.SpawnImg = g.spawnImg
		g.canvas.SpawnImgHover = g.spawnImgHover
		g.canvas.TriangleImg = g.triangleImg
		g.canvas.TriangleImgHover = g.triangleImgHover
		g.canvas.BorderImg = g.borderImg
		g.canvas.LayerTileImgs = g.layerTileImgs
		g.canvas.HighlightPhysics = g.highlightPhysics
		g.canvas.SpawnMode = g.spawnMode
		g.canvas.TriangleMode = g.triangleMode
		g.canvas.TilesetImg = g.tilesetPanel.tilesetImg
		g.canvas.SelectedTile = g.tilesetPanel.selectedTile
		g.canvas.TilesetTileW = g.tilesetPanel.tilesetTileW
		g.canvas.TilesetTileH = g.tilesetPanel.tilesetTileH
		g.canvas.TilesetPath = g.tilesetPanel.tilesetPath
		g.canvas.PushSnapshot = g.pushSnapshot
		g.canvas.PushSnapshotDelta = g.pushSnapshotDelta
		g.canvas.Backgrounds = g.backgrounds
		g.canvas.ControlsText = g.controlsText

		g.canvas.Update(mx, my, panelX, inTilesetPanel)

		// sync back mutated state
		g.canvasZoom = g.canvas.CanvasZoom
		g.canvasOffsetX = g.canvas.CanvasOffsetX
		g.canvasOffsetY = g.canvas.CanvasOffsetY
		g.canvasDragActive = g.canvas.CanvasDragActive
		g.canvasLastMX = g.canvas.CanvasLastMX
		g.canvasLastMY = g.canvas.CanvasLastMY
		g.prevMouse = g.canvas.PrevMouse
		g.dragging = g.canvas.Dragging
		g.rightDragging = g.canvas.RightDragging
		g.paintValue = g.canvas.PaintValue
	}

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

	// Toggle physics highlight (Y)
	if inpututil.IsKeyJustPressed(ebiten.KeyY) {
		g.highlightPhysics = !g.highlightPhysics
	}

	// Undo (Ctrl+Z)
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) && (ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)) {
		g.Undo()
	}

	// Save if S pressed
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if err := g.Save(); err != nil {
			log.Printf("save error: %v", err)
		} else {
			log.Printf("saved to %s", g.filename)
		}
	}

	// Select background image with B (opens native file dialog)
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		if path, err := openBackgroundDialog(); err == nil {
			if path != "" {
				bgPath := normalizeAssetPath(path)
				// attempt to load image from provided path and add as background
				loaded := false
				if b, err := os.ReadFile(path); err == nil {
					if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
						if g.backgrounds != nil {
							g.backgrounds.Add(bgPath, img, g.level, g.cellSize)
						}
						if g.level != nil {
							g.level.Backgrounds = append(g.level.Backgrounds, BackgroundEntry{Path: bgPath, Parallax: 0.5})
						}
						loaded = true
					}
				}
				if !loaded {
					// fallback: try assets/<path> and basename
					if b, err := os.ReadFile(filepath.Join("assets", path)); err == nil {
						if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
							if g.backgrounds != nil {
								g.backgrounds.Add(bgPath, img, g.level, g.cellSize)
							}
							if g.level != nil {
								g.level.Backgrounds = append(g.level.Backgrounds, BackgroundEntry{Path: bgPath, Parallax: 0.5})
							}
							loaded = true
						}
					}
				}
				if !loaded {
					base := filepath.Base(path)
					if b, err := os.ReadFile(filepath.Join("assets", base)); err == nil {
						if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
							if g.backgrounds != nil {
								g.backgrounds.Add(bgPath, img, g.level, g.cellSize)
							}
							if g.level != nil {
								g.level.Backgrounds = append(g.level.Backgrounds, BackgroundEntry{Path: bgPath, Parallax: 0.5})
							}
						}
					}
				}
			}
		} else {
			log.Printf("background dialog error: %v", err)
		}
	}

	return nil
}

// Draw renders the editor.
func (g *Editor) Draw(screen *ebiten.Image) {
	// Draw with canvas transform. Calculate dynamic panel positions from screen size.
	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()
	panelX := screenW - rightPanelWidth
	canvasW := panelX - leftPanelWidth
	if canvasW < 1 {
		canvasW = 1
	}

	// Offscreen canvas to clip drawing within the canvas bounds
	if g.canvasImg == nil {
		g.canvasImg = ebiten.NewImage(canvasW, screenH)
	}

	g.canvasImg.Clear()

	// helper to apply canvas transforms for drawing an image positioned at logical (tx,ty)
	applyCanvas := func(op *ebiten.DrawImageOptions, tx, ty float64) {
		op.GeoM.Translate(tx, ty)                 // position in logical canvas coords
		op.GeoM.Scale(g.canvasZoom, g.canvasZoom) // scale canvas + any tile-scale set earlier
		op.GeoM.Translate(g.canvasOffsetX, g.canvasOffsetY)
	}

	// Draw background layers first (if present)
	if g.level != nil && len(g.level.Backgrounds) > 0 {
		if g.backgrounds != nil {
			g.backgrounds.Draw(g.canvasImg, g.canvasZoom, g.canvasOffsetX, g.canvasOffsetY)
		}
	} else {
		// Draw base empty grid once
		for y := 0; y < g.level.Height; y++ {
			for x := 0; x < g.level.Width; x++ {
				op := &ebiten.DrawImageOptions{}
				applyCanvas(op, float64(x*g.cellSize), float64(y*g.cellSize))
				g.canvasImg.DrawImage(g.emptyImg, op)
			}
		}
	}

	// Draw layers bottom-to-top (only draw tiles; background already drawn)
	for layerIdx := 0; layerIdx < len(g.level.Layers); layerIdx++ {
		tileImg := g.layerTileImgs[layerIdx]

		layer := g.level.Layers[layerIdx]
		for y := 0; y < g.level.Height; y++ {
			for x := 0; x < g.level.Width; x++ {
				idx := y*g.level.Width + x
				val := layer[idx]
				// solid single-color tile
				if val == 1 {
					op := &ebiten.DrawImageOptions{}
					applyCanvas(op, float64(x*g.cellSize), float64(y*g.cellSize))
					g.canvasImg.DrawImage(tileImg, op)
				} else if val == 2 {
					// triangle marker
					if g.triangleImg != nil {
						op := &ebiten.DrawImageOptions{}
						applyCanvas(op, float64(x*g.cellSize), float64(y*g.cellSize))
						g.canvasImg.DrawImage(g.triangleImg, op)
					}
				} else if val >= 3 {
					// tileset-based tile (stored as value = index + 3)
					drawn := false
					entry := (*TilesetEntry)(nil)
					if g.level.TilesetUsage != nil && layerIdx < len(g.level.TilesetUsage) {
						usageLayer := g.level.TilesetUsage[layerIdx]
						if usageLayer != nil && y < len(usageLayer) && x < len(usageLayer[y]) {
							entry = usageLayer[y][x]
						}
					}
					tileW := g.tilesetPanel.tilesetTileW
					tileH := g.tilesetPanel.tilesetTileH
					tileIndex := val - 3
					if entry != nil {
						if entry.TileW > 0 {
							tileW = entry.TileW
						}
						if entry.TileH > 0 {
							tileH = entry.TileH
						}
						tileIndex = entry.Index
					}
					if g.tilesetPanel.tilesetImg != nil && tileW > 0 && tileH > 0 {
						cols := g.tilesetPanel.tilesetImg.Bounds().Dx() / tileW
						rows := g.tilesetPanel.tilesetImg.Bounds().Dy() / tileH
						if cols > 0 && rows > 0 && tileIndex >= 0 {
							col := tileIndex % cols
							row := tileIndex / cols
							sx := col * tileW
							sy := row * tileH
							if sx >= 0 && sy >= 0 && sx+tileW <= g.tilesetPanel.tilesetImg.Bounds().Dx() && sy+tileH <= g.tilesetPanel.tilesetImg.Bounds().Dy() {
								r := image.Rect(sx, sy, sx+tileW, sy+tileH)
								if sub, ok := g.tilesetPanel.tilesetImg.SubImage(r).(*ebiten.Image); ok {
									op := &ebiten.DrawImageOptions{}
									// tile-scale then canvas transform
									op.GeoM.Scale(float64(g.cellSize)/float64(tileW), float64(g.cellSize)/float64(tileH))
									applyCanvas(op, float64(x*g.cellSize), float64(y*g.cellSize))
									g.canvasImg.DrawImage(sub, op)
									drawn = true
								}
							}
						}
					}
					if !drawn {
						op := &ebiten.DrawImageOptions{}
						applyCanvas(op, float64(x*g.cellSize), float64(y*g.cellSize))
						if g.missingImg != nil {
							g.canvasImg.DrawImage(g.missingImg, op)
						}
					}
				}

				// optional physics highlight border for physics-enabled layers (draw for any non-empty tile)
				if val != 0 && g.highlightPhysics && g.level.LayerMeta != nil && layerIdx < len(g.level.LayerMeta) && g.level.LayerMeta[layerIdx].HasPhysics {
					topB := &ebiten.DrawImageOptions{}
					topB.GeoM.Scale(float64(g.cellSize), 1)
					applyCanvas(topB, float64(x*g.cellSize), float64(y*g.cellSize))
					g.canvasImg.DrawImage(g.borderImg, topB)
					bottomB := &ebiten.DrawImageOptions{}
					bottomB.GeoM.Scale(float64(g.cellSize), 1)
					applyCanvas(bottomB, float64(x*g.cellSize), float64(y*g.cellSize+g.cellSize-1))
					g.canvasImg.DrawImage(g.borderImg, bottomB)
					leftB := &ebiten.DrawImageOptions{}
					leftB.GeoM.Scale(1, float64(g.cellSize))
					applyCanvas(leftB, float64(x*g.cellSize), float64(y*g.cellSize))
					g.canvasImg.DrawImage(g.borderImg, leftB)
					rightB := &ebiten.DrawImageOptions{}
					rightB.GeoM.Scale(1, float64(g.cellSize))
					applyCanvas(rightB, float64(x*g.cellSize+g.cellSize-1), float64(y*g.cellSize))
					g.canvasImg.DrawImage(g.borderImg, rightB)
				}
			}
		}
	}

	// Hover highlight (draw on top) using canvas transforms
	mx, my := ebiten.CursorPosition()
	if g.level != nil {
		if cx, cy, ok := g.screenToCanvasPoint(mx, my, panelX); ok {
			gx := int(math.Floor(cx / float64(g.cellSize)))
			gy := int(math.Floor(cy / float64(g.cellSize)))
			if gx >= 0 && gy >= 0 && gx < g.level.Width && gy < g.level.Height {
				hop := &ebiten.DrawImageOptions{}
				applyCanvas(hop, float64(gx*g.cellSize), float64(gy*g.cellSize))
				if g.spawnMode {
					g.canvasImg.DrawImage(g.spawnImgHover, hop)
				} else if g.triangleMode {
					g.canvasImg.DrawImage(g.triangleImgHover, hop)
				} else {
					g.canvasImg.DrawImage(g.hoverImg, hop)
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
			applyCanvas(sop, float64(sx*g.cellSize), float64(sy*g.cellSize))
			g.canvasImg.DrawImage(g.spawnImg, sop)
		}
	}

	// Draw controls text inside canvas by syncing Editor->Canvas and delegating
	if g.canvas != nil {
		// ensure canvas left panel width matches UI constant
		g.canvas.LeftPanelW = leftPanelWidth
		g.canvas.CanvasZoom = g.canvasZoom
		g.canvas.CanvasOffsetX = g.canvasOffsetX
		g.canvas.CanvasOffsetY = g.canvasOffsetY
		g.canvas.CanvasDragActive = g.canvasDragActive
		g.canvas.CanvasLastMX = g.canvasLastMX
		g.canvas.CanvasLastMY = g.canvasLastMY
		g.canvas.PrevMouse = g.prevMouse
		g.canvas.Dragging = g.dragging
		g.canvas.RightDragging = g.rightDragging
		g.canvas.PaintValue = g.paintValue
		g.canvas.Level = g.level
		g.canvas.CellSize = g.cellSize
		g.canvas.CurrentLayer = g.currentLayer
		g.canvas.EmptyImg = g.emptyImg
		g.canvas.HoverImg = g.hoverImg
		g.canvas.MissingImg = g.missingImg
		g.canvas.SpawnImg = g.spawnImg
		g.canvas.SpawnImgHover = g.spawnImgHover
		g.canvas.TriangleImg = g.triangleImg
		g.canvas.TriangleImgHover = g.triangleImgHover
		g.canvas.BorderImg = g.borderImg
		g.canvas.LayerTileImgs = g.layerTileImgs
		g.canvas.HighlightPhysics = g.highlightPhysics
		g.canvas.SpawnMode = g.spawnMode
		g.canvas.TriangleMode = g.triangleMode
		g.canvas.PushSnapshot = g.pushSnapshot
		g.canvas.Backgrounds = g.backgrounds
		g.canvas.ControlsText = g.controlsText

		g.canvas.ControlsText.Draw(g.canvasImg, g.canvas)
	}

	// Draw canvas onto the screen within the panel bounds
	canvasOp := &ebiten.DrawImageOptions{}
	canvasOp.GeoM.Translate(float64(leftPanelWidth), 0)
	screen.DrawImage(g.canvasImg, canvasOp)

	// Draw right-side panel for tileset and assets (panelX computed above)
	// keep tileset panel anchored to right
	g.tilesetPanel.X = panelX + 8

	g.tilesetPanel.Draw(screen, panelX)
	g.entityPanel.Draw(screen)
}

func (g *Editor) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	// Use the full available outside size so the editor fills the window.
	return outsideWidth, outsideHeight
}

func (g *Editor) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("Layout called; use LayoutF instead")
}

// ControlsText draws help text inside the canvas.
func (ct ControlsText) Draw(canvas *ebiten.Image, c *Canvas) {
	curMeta := LayerMeta{}
	layerIdx := 0
	if c.Level != nil && c.Level.LayerMeta != nil && c.CurrentLayer < len(c.Level.LayerMeta) {
		layerIdx = c.CurrentLayer
		curMeta = c.Level.LayerMeta[layerIdx]
	}
	filename := ""
	backgrounds := 0
	spawnX, spawnY := 0, 0
	if c.Level != nil {
		filename = ""
		backgrounds = len(c.Level.Backgrounds)
		spawnX = c.Level.SpawnX
		spawnY = c.Level.SpawnY
	}
	instr := fmt.Sprintf("Left-click: toggle tile   S: save   Q/E: cycle layers   N: new layer   H: toggle physics   Y: highlight physics   P: place spawn   T: triangle mode  Y: highlight physics  B: add background   File: %s\nW=%d H=%d Cell=%d Layer=%d has_physics=%v color=%s spawn=(%d,%d) spawnMode=%v triangleMode=%v backgrounds=%d",
		filename, func() int {
			if c.Level != nil {
				return c.Level.Width
			}
			return 0
		}(), func() int {
			if c.Level != nil {
				return c.Level.Height
			}
			return 0
		}(), c.CellSize, layerIdx, curMeta.HasPhysics, curMeta.Color, spawnX, spawnY, c.SpawnMode, c.TriangleMode, backgrounds)
	ebitenutil.DebugPrintAt(canvas, instr, ct.X, ct.Y)
}
