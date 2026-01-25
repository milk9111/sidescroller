package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
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
	// TilesetUsage stores per-layer, per-cell tileset metadata when a tileset tile is used.
	TilesetUsage [][][]*TilesetEntry `json:"tileset_usage,omitempty"`
}

// TilesetEntry records which tileset file and tile index plus tile size used for a cell.
type TilesetEntry struct {
	Path  string `json:"path"`
	Index int    `json:"index"`
	TileW int    `json:"tile_w"`
	TileH int    `json:"tile_h"`
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
	// drag paint state
	dragging   bool
	paintValue int
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

	// tileset support
	tilesetImg       *ebiten.Image
	tilesetPath      string
	tilesetTileW     int
	tilesetTileH     int
	tilesetCols      int
	selectedTile     int // 0-based index
	assetList        []string
	highlightPhysics bool
	borderImg        *ebiten.Image
	// tileset panel UI state (draggable, zoomable)
	tilesetPanelX     int
	tilesetPanelY     int
	tilesetPanelW     int
	tilesetPanelH     int
	tilesetZoom       float64
	tilesetOffsetX    float64
	tilesetOffsetY    float64
	tilesetDragActive bool
	tilesetLastMX     int
	tilesetLastMY     int
	tilesetHover      int // hovered tile index, -1 none
	panelBgImg        *ebiten.Image
	hoverBorderImg    *ebiten.Image
	selectBorderImg   *ebiten.Image
	prevRight         bool
	// missing image drawn when a tileset subimage can't be extracted
	missingImg *ebiten.Image
	// undo stack: stores past copies of Layers for undo
	undoStack [][][]int
	maxUndo   int
}

const (
	// Increase base width to accomodate tileset panel to the right
	baseWidthEditor  = 40*32 + 220 // 1280 + 220 = 1500
	baseHeightEditor = 23 * 32     // 736
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

	// populate embedded asset list from assets/ (if available)
	if entries, err := os.ReadDir("assets"); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				name := e.Name()
				if len(name) > 4 && (name[len(name)-4:] == ".png" || name[len(name)-4:] == ".PNG") {
					eg.assetList = append(eg.assetList, name)
				}
			}
		}
	}

	eg.selectedTile = -1
	eg.maxUndo = 100

	// tileset panel defaults
	sideWidth := 220
	panelX := baseWidthEditor - sideWidth
	eg.tilesetPanelX = panelX + 8
	// place panel below asset list area (approx)
	eg.tilesetPanelY = 8 + len(eg.assetList)*18 + 8
	eg.tilesetPanelW = 184
	eg.tilesetPanelH = 220
	eg.tilesetZoom = 1.0
	eg.tilesetOffsetX = 0
	eg.tilesetOffsetY = 0
	eg.tilesetHover = -1

	// panel background (1x1) and hover/select borders
	bg := ebiten.NewImage(1, 1)
	bg.Fill(color.RGBA{0x00, 0x00, 0x00, 0x44})
	eg.panelBgImg = bg
	hb := ebiten.NewImage(1, 1)
	hb.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	eg.hoverBorderImg = hb
	sb := ebiten.NewImage(1, 1)
	sb.Fill(color.RGBA{0xff, 0xd7, 0x00, 0xff})
	eg.selectBorderImg = sb

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

	// Right-side panel for tileset and assets (split into file list + a draggable, zoomable tileset panel)
	// asset list area (click to load an asset)
	listStartY := 8
	lineH := 18
	if pressed && !g.prevMouse {
		for i, name := range g.assetList {
			y0 := listStartY + i*lineH
			if my >= y0 && my < y0+16 {
				if b, err := os.ReadFile(filepath.Join("assets", name)); err == nil {
					if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
						g.tilesetImg = ebiten.NewImageFromImage(img)
						// default tiles size to cellSize unless already specified
						if g.tilesetTileW == 0 {
							g.tilesetTileW = g.cellSize
						}
						if g.tilesetTileH == 0 {
							g.tilesetTileH = g.cellSize
						}
						if g.tilesetTileW > 0 {
							g.tilesetCols = g.tilesetImg.Bounds().Dx() / g.tilesetTileW
						}
						g.tilesetPath = name
						g.selectedTile = 0
					}
				}
				break
			}
		}
	}

	// tileset panel interactions: hover, left-click select, right-drag pan, mouse-wheel zoom
	if g.tilesetImg != nil {
		// detect if cursor is inside the tileset panel
		inTilesetPanel := mx >= g.tilesetPanelX && mx < g.tilesetPanelX+g.tilesetPanelW && my >= g.tilesetPanelY && my < g.tilesetPanelY+g.tilesetPanelH

		// wheel zoom (centered on mouse)
		if inTilesetPanel {
			_, wy := ebiten.Wheel()
			if wy != 0 {
				// compute local tile-space coordinate before zoom
				localX := (float64(mx) - float64(g.tilesetPanelX) - 8 - g.tilesetOffsetX) / g.tilesetZoom
				localY := (float64(my) - float64(g.tilesetPanelY) - 8 - g.tilesetOffsetY) / g.tilesetZoom
				var factor float64
				if wy > 0 {
					factor = 1.1
				} else {
					factor = 1.0 / 1.1
				}
				newZoom := g.tilesetZoom * factor
				if newZoom < 0.25 {
					newZoom = 0.25
				}
				if newZoom > 4.0 {
					newZoom = 4.0
				}
				g.tilesetZoom = newZoom
				// recompute offset so point under cursor stays fixed
				g.tilesetOffsetX = float64(mx) - float64(g.tilesetPanelX) - 8 - localX*g.tilesetZoom
				g.tilesetOffsetY = float64(my) - float64(g.tilesetPanelY) - 8 - localY*g.tilesetZoom
			}
		}

		// right-button drag to pan
		rPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
		if rPressed {
			if !g.tilesetDragActive && inTilesetPanel {
				g.tilesetDragActive = true
				g.tilesetLastMX = mx
				g.tilesetLastMY = my
			}
			if g.tilesetDragActive {
				dx := mx - g.tilesetLastMX
				dy := my - g.tilesetLastMY
				g.tilesetOffsetX += float64(dx)
				g.tilesetOffsetY += float64(dy)
				g.tilesetLastMX = mx
				g.tilesetLastMY = my
			}
		} else {
			g.tilesetDragActive = false
		}

		// clamp offsets so tileset content cannot be dragged completely out of the panel
		if g.tilesetTileW > 0 && g.tilesetTileH > 0 {
			cols := g.tilesetImg.Bounds().Dx() / g.tilesetTileW
			rows := g.tilesetImg.Bounds().Dy() / g.tilesetTileH
			contentW := float64(cols) * float64(g.tilesetTileW) * g.tilesetZoom
			contentH := float64(rows) * float64(g.tilesetTileH) * g.tilesetZoom
			innerW := float64(g.tilesetPanelW - 16)
			innerH := float64(g.tilesetPanelH - 16)
			// min offset so right/bottom edges still cover panel
			minX := math.Min(0, innerW-contentW)
			minY := math.Min(0, innerH-contentH)
			if g.tilesetOffsetX < minX {
				g.tilesetOffsetX = minX
			}
			if g.tilesetOffsetY < minY {
				g.tilesetOffsetY = minY
			}
			if g.tilesetOffsetX > 0 {
				g.tilesetOffsetX = 0
			}
			if g.tilesetOffsetY > 0 {
				g.tilesetOffsetY = 0
			}
		}

		// compute hover tile under mouse (even without clicks)
		g.tilesetHover = -1
		if inTilesetPanel && g.tilesetTileW > 0 && g.tilesetTileH > 0 {
			localX := (float64(mx) - float64(g.tilesetPanelX) - 8 - g.tilesetOffsetX) / (float64(g.tilesetTileW) * g.tilesetZoom)
			localY := (float64(my) - float64(g.tilesetPanelY) - 8 - g.tilesetOffsetY) / (float64(g.tilesetTileH) * g.tilesetZoom)
			if localX >= 0 && localY >= 0 {
				col := int(math.Floor(localX))
				row := int(math.Floor(localY))
				cols := g.tilesetImg.Bounds().Dx() / g.tilesetTileW
				rows := g.tilesetImg.Bounds().Dy() / g.tilesetTileH
				if col >= 0 && row >= 0 && col < cols && row < rows {
					g.tilesetHover = row*cols + col
				}
			}
		}

		// left-click selection in tileset panel
		if pressed && !g.prevMouse && g.tilesetHover >= 0 {
			g.selectedTile = g.tilesetHover
		}
	}

	// adjust tileset tile size with keys: K (increase), J (decrease)
	if g.tilesetImg != nil {
		if inpututil.IsKeyJustPressed(ebiten.KeyK) {
			g.tilesetTileW += 16
			g.tilesetTileH += 16
			if g.tilesetTileW < 1 {
				g.tilesetTileW = 1
			}
			if g.tilesetTileH < 1 {
				g.tilesetTileH = 1
			}
			if g.tilesetTileW > 0 {
				g.tilesetCols = g.tilesetImg.Bounds().Dx() / g.tilesetTileW
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyJ) {
			if g.tilesetTileW > 16 {
				g.tilesetTileW -= 16
			} else {
				g.tilesetTileW = 1
			}
			if g.tilesetTileH > 16 {
				g.tilesetTileH -= 16
			} else {
				g.tilesetTileH = 1
			}
			if g.tilesetTileW > 0 {
				g.tilesetCols = g.tilesetImg.Bounds().Dx() / g.tilesetTileW
			}
		}
	}

	// Right-click erase: immediate erase on right-button click inside canvas
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if mx >= 0 && my >= 0 && mx < w && my < h {
			cx := mx / g.cellSize
			cy := my / g.cellSize
			idx := cy*g.level.Width + cx
			if idx >= 0 && idx < g.level.Width*g.level.Height {
				if g.level.Layers == nil || len(g.level.Layers) == 0 {
					g.level.Layers = make([][]int, 1)
					g.level.Layers[0] = make([]int, g.level.Width*g.level.Height)
				}
				g.pushSnapshot()
				layer := g.level.Layers[g.currentLayer]
				layer[idx] = 0
				g.level.Layers[g.currentLayer] = layer
			}
		}
	}

	// Handle initial press: determine paintValue and start dragging (unless spawnMode)
	if pressed && !g.prevMouse {
		if !g.spawnMode && mx >= 0 && my >= 0 && mx < w && my < h {
			cx := mx / g.cellSize
			cy := my / g.cellSize
			idx := cy*g.level.Width + cx
			if idx >= 0 && idx < g.level.Width*g.level.Height {
				// ensure Layers exists
				if g.level.Layers == nil || len(g.level.Layers) == 0 {
					g.level.Layers = make([][]int, 1)
					g.level.Layers[0] = make([]int, g.level.Width*g.level.Height)
				}
				// snapshot before making an edit for undo
				g.pushSnapshot()

				layer := g.level.Layers[g.currentLayer]
				// triangle placement mode (only allowed on physics-enabled layers)
				canTriangle := false
				if g.level.LayerMeta != nil && g.currentLayer < len(g.level.LayerMeta) {
					canTriangle = g.level.LayerMeta[g.currentLayer].HasPhysics
				}
				// decide paintValue: if a tileset is loaded and a tile is selected, use that tile index (offset by 3 to avoid colliding with reserved values)
				if g.tilesetImg != nil && g.selectedTile >= 0 {
					g.paintValue = g.selectedTile + 3
				} else if g.triangleMode && canTriangle {
					if layer[idx] == 2 {
						g.paintValue = 0
					} else {
						g.paintValue = 2
					}
				} else {
					if layer[idx] == 0 {
						g.paintValue = 1
					} else {
						g.paintValue = 0
					}
				}
				// start dragging and apply immediately
				g.dragging = true
				layer[idx] = g.paintValue
				g.level.Layers[g.currentLayer] = layer
			}
		}
	} else if pressed && g.prevMouse && g.dragging && !g.spawnMode {
		// dragging: apply paintValue to hovered cell
		if mx >= 0 && my >= 0 && mx < w && my < h {
			cx := mx / g.cellSize
			cy := my / g.cellSize
			idx := cy*g.level.Width + cx
			if idx >= 0 && idx < g.level.Width*g.level.Height {
				if g.level.Layers == nil || len(g.level.Layers) == 0 {
					g.level.Layers = make([][]int, 1)
					g.level.Layers[0] = make([]int, g.level.Width*g.level.Height)
				}
				layer := g.level.Layers[g.currentLayer]
				if layer[idx] != g.paintValue {
					layer[idx] = g.paintValue
					g.level.Layers[g.currentLayer] = layer
				}
			}
		}
	}

	// end dragging on mouse release
	if !pressed && g.prevMouse {
		g.dragging = false
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
				val := layer[idx]
				// solid single-color tile
				if val == 1 {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
					screen.DrawImage(tileImg, op)
				} else if val == 2 {
					// triangle marker
					if g.triangleImg != nil {
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
						screen.DrawImage(g.triangleImg, op)
					}
				} else if val >= 3 {
					// tileset-based tile (stored as value = index + 3)
					drawn := false
					if g.tilesetImg != nil && g.tilesetTileW > 0 && g.tilesetTileH > 0 {
						tileIndex := val - 3
						cols := g.tilesetImg.Bounds().Dx() / g.tilesetTileW
						rows := g.tilesetImg.Bounds().Dy() / g.tilesetTileH
						if cols > 0 && rows > 0 && tileIndex >= 0 {
							col := tileIndex % cols
							row := tileIndex / cols
							sx := col * g.tilesetTileW
							sy := row * g.tilesetTileH
							if sx >= 0 && sy >= 0 && sx+g.tilesetTileW <= g.tilesetImg.Bounds().Dx() && sy+g.tilesetTileH <= g.tilesetImg.Bounds().Dy() {
								r := image.Rect(sx, sy, sx+g.tilesetTileW, sy+g.tilesetTileH)
								if sub, ok := g.tilesetImg.SubImage(r).(*ebiten.Image); ok {
									op := &ebiten.DrawImageOptions{}
									scaleX := float64(g.cellSize) / float64(g.tilesetTileW)
									scaleY := float64(g.cellSize) / float64(g.tilesetTileH)
									op.GeoM.Scale(scaleX, scaleY)
									op.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
									screen.DrawImage(sub, op)
									drawn = true
								}
							}
						}
					}
					if !drawn {
						op := &ebiten.DrawImageOptions{}
						op.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
						if g.missingImg != nil {
							screen.DrawImage(g.missingImg, op)
						}
					}
				}

				// optional physics highlight border for physics-enabled layers (draw for any non-empty tile)
				if val != 0 && g.highlightPhysics && g.level.LayerMeta != nil && layerIdx < len(g.level.LayerMeta) && g.level.LayerMeta[layerIdx].HasPhysics {
					topB := &ebiten.DrawImageOptions{}
					topB.GeoM.Scale(float64(g.cellSize), 1)
					topB.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
					screen.DrawImage(g.borderImg, topB)
					bottomB := &ebiten.DrawImageOptions{}
					bottomB.GeoM.Scale(float64(g.cellSize), 1)
					bottomB.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize+g.cellSize-1))
					screen.DrawImage(g.borderImg, bottomB)
					leftB := &ebiten.DrawImageOptions{}
					leftB.GeoM.Scale(1, float64(g.cellSize))
					leftB.GeoM.Translate(float64(x*g.cellSize), float64(y*g.cellSize))
					screen.DrawImage(g.borderImg, leftB)
					rightB := &ebiten.DrawImageOptions{}
					rightB.GeoM.Scale(1, float64(g.cellSize))
					rightB.GeoM.Translate(float64(x*g.cellSize+g.cellSize-1), float64(y*g.cellSize))
					screen.DrawImage(g.borderImg, rightB)
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
	instr := fmt.Sprintf("Left-click: toggle tile   S: save   Q/E: cycle layers   N: new layer   H: toggle physics   Y: highlight physics   P: place spawn   T: triangle mode   File: %s\nW=%d H=%d Cell=%d Layer=%d has_physics=%v color=%s spawn=(%d,%d) spawnMode=%v triangleMode=%v",
		g.filename, g.level.Width, g.level.Height, g.cellSize, g.currentLayer, curMeta.HasPhysics, curMeta.Color, g.level.SpawnX, g.level.SpawnY, g.spawnMode, g.triangleMode)
	ebitenutil.DebugPrint(screen, instr)

	// Draw right-side panel for tileset and assets
	sideWidth := 220
	panelX := baseWidthEditor - sideWidth
	// asset list
	y := 8
	for i, name := range g.assetList {
		ebitenutil.DebugPrintAt(screen, name, panelX+8, y+i*18)
	}
	// tileset panel (draggable, zoomable)
	if g.tilesetImg != nil {
		// ensure panel bounds exist
		// draw panel background
		bgOp := &ebiten.DrawImageOptions{}
		bgOp.GeoM.Scale(float64(g.tilesetPanelW), float64(g.tilesetPanelH))
		bgOp.GeoM.Translate(float64(g.tilesetPanelX), float64(g.tilesetPanelY))
		screen.DrawImage(g.panelBgImg, bgOp)

		cols := 1
		if g.tilesetTileW > 0 {
			cols = g.tilesetImg.Bounds().Dx() / g.tilesetTileW
		}
		rows := 1
		if g.tilesetTileH > 0 {
			rows = g.tilesetImg.Bounds().Dy() / g.tilesetTileH
		}
		tileW := float64(g.tilesetTileW) * g.tilesetZoom
		tileH := float64(g.tilesetTileH) * g.tilesetZoom
		// draw tiles
		for ry := 0; ry < rows; ry++ {
			for rx := 0; rx < cols; rx++ {
				idx := ry*cols + rx
				sx := rx * g.tilesetTileW
				sy := ry * g.tilesetTileH
				r := image.Rect(sx, sy, sx+g.tilesetTileW, sy+g.tilesetTileH)
				sub := g.tilesetImg.SubImage(r).(*ebiten.Image)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(g.tilesetZoom, g.tilesetZoom)
				dx := float64(g.tilesetPanelX+8) + g.tilesetOffsetX + float64(rx)*tileW
				dy := float64(g.tilesetPanelY+8) + g.tilesetOffsetY + float64(ry)*tileH
				op.GeoM.Translate(dx, dy)
				screen.DrawImage(sub, op)

				// hover border
				if g.tilesetHover == idx {
					hbOp := &ebiten.DrawImageOptions{}
					hbOp.GeoM.Scale(tileW, 1)
					hbOp.GeoM.Translate(dx, dy)
					screen.DrawImage(g.hoverBorderImg, hbOp)
					hbOp2 := &ebiten.DrawImageOptions{}
					hbOp2.GeoM.Scale(tileW, 1)
					hbOp2.GeoM.Translate(dx, dy+tileH-1)
					screen.DrawImage(g.hoverBorderImg, hbOp2)
					hbOp3 := &ebiten.DrawImageOptions{}
					hbOp3.GeoM.Scale(1, tileH)
					hbOp3.GeoM.Translate(dx, dy)
					screen.DrawImage(g.hoverBorderImg, hbOp3)
					hbOp4 := &ebiten.DrawImageOptions{}
					hbOp4.GeoM.Scale(1, tileH)
					hbOp4.GeoM.Translate(dx+tileW-1, dy)
					screen.DrawImage(g.hoverBorderImg, hbOp4)
				}

				// selected border
				if g.selectedTile == idx {
					sbOp := &ebiten.DrawImageOptions{}
					sbOp.GeoM.Scale(tileW, 1)
					sbOp.GeoM.Translate(dx, dy)
					screen.DrawImage(g.selectBorderImg, sbOp)
					sbOp2 := &ebiten.DrawImageOptions{}
					sbOp2.GeoM.Scale(tileW, 1)
					sbOp2.GeoM.Translate(dx, dy+tileH-1)
					screen.DrawImage(g.selectBorderImg, sbOp2)
					sbOp3 := &ebiten.DrawImageOptions{}
					sbOp3.GeoM.Scale(1, tileH)
					sbOp3.GeoM.Translate(dx, dy)
					screen.DrawImage(g.selectBorderImg, sbOp3)
					sbOp4 := &ebiten.DrawImageOptions{}
					sbOp4.GeoM.Scale(1, tileH)
					sbOp4.GeoM.Translate(dx+tileW-1, dy)
					screen.DrawImage(g.selectBorderImg, sbOp4)
				}
			}
		}
	}

	// show current tileset settings
	infoY := baseHeightEditor - 80
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tileset: %s", g.tilesetPath), panelX+8, infoY)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("TileSize: %dx%d", g.tilesetTileW, g.tilesetTileH), panelX+8, infoY+18)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Highlight physics (Y): %v", g.highlightPhysics), panelX+8, infoY+36)
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
	// build TilesetUsage: per-layer 2D arrays of tileset info for cells that use a tileset
	if g.level != nil {
		usage := make([][][]*TilesetEntry, len(g.level.Layers))
		for li := range g.level.Layers {
			layer := g.level.Layers[li]
			rows := make([][]*TilesetEntry, g.level.Height)
			for y := 0; y < g.level.Height; y++ {
				row := make([]*TilesetEntry, g.level.Width)
				for x := 0; x < g.level.Width; x++ {
					idx := y*g.level.Width + x
					if idx >= 0 && idx < len(layer) {
						v := layer[idx]
						if v >= 3 && g.tilesetPath != "" && g.tilesetTileW > 0 && g.tilesetTileH > 0 {
							row[x] = &TilesetEntry{Path: g.tilesetPath, Index: v - 3, TileW: g.tilesetTileW, TileH: g.tilesetTileH}
						} else {
							row[x] = nil
						}
					}
				}
				rows[y] = row
			}
			usage[li] = rows
		}
		g.level.TilesetUsage = usage
	}

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

	// If the saved level contains TilesetUsage metadata, try to open the first referenced tileset
	if g.level != nil && g.level.TilesetUsage != nil {
		found := false
		for li := range g.level.TilesetUsage {
			layerUsage := g.level.TilesetUsage[li]
			if layerUsage == nil {
				continue
			}
			for y := 0; y < g.level.Height && !found; y++ {
				for x := 0; x < g.level.Width; x++ {
					if y < len(layerUsage) && x < len(layerUsage[y]) {
						entry := layerUsage[y][x]
						if entry != nil && entry.Path != "" {
							// attempt to load tileset from assets/<path>
							if b, err := os.ReadFile(filepath.Join("assets", entry.Path)); err == nil {
								if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
									g.tilesetImg = ebiten.NewImageFromImage(img)
									g.tilesetPath = entry.Path
									g.tilesetTileW = entry.TileW
									g.tilesetTileH = entry.TileH
									if g.tilesetTileW > 0 {
										g.tilesetCols = g.tilesetImg.Bounds().Dx() / g.tilesetTileW
									}
									g.selectedTile = entry.Index
									found = true
									break
								}
							}
						}
					}
				}
			}
			if found {
				break
			}
		}
	}

	return nil
}

// pushSnapshot stores a deep copy of the current Layers for undo.
func (g *Editor) pushSnapshot() {
	if g.level == nil || g.level.Layers == nil {
		return
	}
	// deep copy layers
	copyLayers := make([][]int, len(g.level.Layers))
	for i := range g.level.Layers {
		layer := g.level.Layers[i]
		lcopy := make([]int, len(layer))
		copy(lcopy, layer)
		copyLayers[i] = lcopy
	}
	g.undoStack = append(g.undoStack, copyLayers)
	if len(g.undoStack) > g.maxUndo {
		// drop oldest
		g.undoStack = g.undoStack[1:]
	}
}

// Undo restores the last snapshot if available.
func (g *Editor) Undo() {
	n := len(g.undoStack)
	if n == 0 {
		return
	}
	snap := g.undoStack[n-1]
	g.undoStack = g.undoStack[:n-1]
	// apply snapshot
	g.level.Layers = make([][]int, len(snap))
	for i := range snap {
		layer := snap[i]
		lcopy := make([]int, len(layer))
		copy(lcopy, layer)
		g.level.Layers[i] = lcopy
	}
	if g.currentLayer >= len(g.level.Layers) {
		g.currentLayer = len(g.level.Layers) - 1
		if g.currentLayer < 0 {
			g.currentLayer = 0
		}
	}
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
