package main

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Canvas encapsulates the editor's main drawing canvas and interactions.
type Canvas struct {
	// UI/layout
	LeftPanelW int

	// mutable canvas transform + drag state
	CanvasZoom       float64
	CanvasOffsetX    float64
	CanvasOffsetY    float64
	CanvasDragActive bool
	CanvasLastMX     int
	CanvasLastMY     int

	// interaction state
	PrevMouse     bool
	Dragging      bool
	RightDragging bool
	PaintValue    int

	// level/model
	Level        *Level
	CellSize     int
	CurrentLayer int

	// tileset & selection
	TilesetImg   *ebiten.Image
	SelectedTile int
	TilesetTileW int
	TilesetTileH int
	TilesetPath  string

	// images and rendering aids
	EmptyImg         *ebiten.Image
	HoverImg         *ebiten.Image
	MissingImg       *ebiten.Image
	SpawnImg         *ebiten.Image
	SpawnImgHover    *ebiten.Image
	TriangleImg      *ebiten.Image
	TriangleImgHover *ebiten.Image
	BorderImg        *ebiten.Image
	LayerTileImgs    []*ebiten.Image
	HighlightPhysics bool

	// modes & helpers
	SpawnMode    bool
	TriangleMode bool

	// callbacks / managers
	PushSnapshot      func(layer int, indices []int)
	PushSnapshotDelta func(ld LayerDelta)

	// pending delta coalescing during a drag
	PendingDeltaActive bool
	PendingDeltaLayer  int
	PendingDeltaMap    map[int]int
	Backgrounds        *Background

	// small UI text renderer
	ControlsText ControlsText
}

func NewCanvas() *Canvas { return &Canvas{} }

// Update handles input and state changes related to the canvas (pan/zoom/paint/erase).
// Returns nothing; it mutates the Editor state directly.
func (c *Canvas) Update(mx, my, panelX int, inTilesetPanel bool) {
	// helper: transform screen coords to canvas-local (unzoomed) coords and test inside canvas
	screenToCanvas := func(sx, sy int) (float64, float64, bool) {
		if sx < c.LeftPanelW || sx >= panelX {
			return 0, 0, false
		}
		// local pixel inside canvas (relative to left panel)
		lx := float64(sx - c.LeftPanelW)
		ly := float64(sy)
		// map through pan/zoom
		if c.CanvasZoom == 0 {
			c.CanvasZoom = 1.0
		}
		cx := (lx - c.CanvasOffsetX) / c.CanvasZoom
		cy := (ly - c.CanvasOffsetY) / c.CanvasZoom
		return cx, cy, true
	}

	// spawn placement (edge)
	if c.SpawnMode && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if cx, cy, ok := screenToCanvas(mx, my); ok && c.Level != nil {
			gx := int(math.Floor(cx / float64(c.CellSize)))
			gy := int(math.Floor(cy / float64(c.CellSize)))
			if gx >= 0 && gy >= 0 && gx < c.Level.Width && gy < c.Level.Height {
				c.Level.SpawnX = gx
				c.Level.SpawnY = gy
			}
		}
	}

	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	// Canvas zoom with mouse wheel (when cursor over canvas area)
	if mx >= c.LeftPanelW && mx < panelX {
		_, wy := ebiten.Wheel()
		if wy != 0 {
			// local canvas coordinate before zoom
			localX, localY, _ := screenToCanvas(mx, my)
			var factor float64
			if wy > 0 {
				factor = 1.1
			} else {
				factor = 1.0 / 1.1
			}
			newZoom := c.CanvasZoom * factor
			if newZoom < 0.25 {
				newZoom = 0.25
			}
			if newZoom > 8.0 {
				newZoom = 8.0
			}
			c.CanvasZoom = newZoom
			// recompute offset so point under cursor stays fixed
			c.CanvasOffsetX = float64(mx-c.LeftPanelW) - localX*c.CanvasZoom
			c.CanvasOffsetY = float64(my) - localY*c.CanvasZoom
		}
	}

	// Middle-button drag to pan canvas
	mPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
	if mPressed {
		if !c.CanvasDragActive {
			c.CanvasDragActive = true
			c.CanvasLastMX = mx
			c.CanvasLastMY = my
		}
		if c.CanvasDragActive {
			dx := mx - c.CanvasLastMX
			dy := my - c.CanvasLastMY
			c.CanvasOffsetX += float64(dx)
			c.CanvasOffsetY += float64(dy)
			c.CanvasLastMX = mx
			c.CanvasLastMY = my
		}
	} else {
		c.CanvasDragActive = false
	}

	// Right-click erase: allow dragging to clear multiple tiles inside canvas
	rPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	if !inTilesetPanel && c.Level != nil {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			if cx, cy, ok := screenToCanvas(mx, my); ok {
				gx := int(math.Floor(cx / float64(c.CellSize)))
				gy := int(math.Floor(cy / float64(c.CellSize)))
				if gx >= 0 && gy >= 0 && gx < c.Level.Width && gy < c.Level.Height {
					idx := gy*c.Level.Width + gx
					if c.Level.Layers == nil || len(c.Level.Layers) == 0 {
						c.Level.Layers = make([][]int, 1)
						c.Level.Layers[0] = make([]int, c.Level.Width*c.Level.Height)
					}
					if c.Level.TilesetUsage != nil && c.CurrentLayer < len(c.Level.TilesetUsage) && c.Level.TilesetUsage[c.CurrentLayer] != nil && gy < len(c.Level.TilesetUsage[c.CurrentLayer]) && gx < len(c.Level.TilesetUsage[c.CurrentLayer][gy]) {
						c.Level.TilesetUsage[c.CurrentLayer][gy][gx] = nil
					}
					// begin pending delta collection
					c.PendingDeltaActive = true
					c.PendingDeltaLayer = c.CurrentLayer
					if c.PendingDeltaMap == nil {
						c.PendingDeltaMap = make(map[int]int)
					}
					// record previous value
					layer := c.Level.Layers[c.CurrentLayer]
					if _, seen := c.PendingDeltaMap[idx]; !seen {
						c.PendingDeltaMap[idx] = layer[idx]
					}
					layer[idx] = 0
					c.Level.Layers[c.CurrentLayer] = layer
					c.RightDragging = true
				}
			}
		}
		if rPressed && c.RightDragging {
			if cx, cy, ok := screenToCanvas(mx, my); ok {
				gx := int(math.Floor(cx / float64(c.CellSize)))
				gy := int(math.Floor(cy / float64(c.CellSize)))
				if gx >= 0 && gy >= 0 && gx < c.Level.Width && gy < c.Level.Height {
					idx := gy*c.Level.Width + gx
					if c.Level.Layers == nil || len(c.Level.Layers) == 0 {
						c.Level.Layers = make([][]int, 1)
						c.Level.Layers[0] = make([]int, c.Level.Width*c.Level.Height)
					}
					if c.Level.TilesetUsage != nil && c.CurrentLayer < len(c.Level.TilesetUsage) && c.Level.TilesetUsage[c.CurrentLayer] != nil && gy < len(c.Level.TilesetUsage[c.CurrentLayer]) && gx < len(c.Level.TilesetUsage[c.CurrentLayer][gy]) {
						c.Level.TilesetUsage[c.CurrentLayer][gy][gx] = nil
					}
					layer := c.Level.Layers[c.CurrentLayer]
					if layer[idx] != 0 {
						if c.PendingDeltaMap == nil {
							c.PendingDeltaMap = make(map[int]int)
						}
						if _, seen := c.PendingDeltaMap[idx]; !seen {
							c.PendingDeltaMap[idx] = layer[idx]
						}
						layer[idx] = 0
						c.Level.Layers[c.CurrentLayer] = layer
					}
				}
			}
		}
	}

	// Handle initial press: determine paintValue and start dragging (unless spawnMode)
	if pressed && !c.PrevMouse {
		if !c.SpawnMode && c.Level != nil {
			if cx, cy, ok := screenToCanvas(mx, my); ok {
				gx := int(math.Floor(cx / float64(c.CellSize)))
				gy := int(math.Floor(cy / float64(c.CellSize)))
				if gx >= 0 && gy >= 0 && gx < c.Level.Width && gy < c.Level.Height {
					idx := gy*c.Level.Width + gx
					// ensure Layers exists
					if c.Level.Layers == nil || len(c.Level.Layers) == 0 {
						c.Level.Layers = make([][]int, 1)
						c.Level.Layers[0] = make([]int, c.Level.Width*c.Level.Height)
					}
					// begin pending delta collection for undo
					c.PendingDeltaActive = true
					c.PendingDeltaLayer = c.CurrentLayer
					if c.PendingDeltaMap == nil {
						c.PendingDeltaMap = make(map[int]int)
					}
					layer := c.Level.Layers[c.CurrentLayer]
					if _, seen := c.PendingDeltaMap[idx]; !seen {
						c.PendingDeltaMap[idx] = layer[idx]
					}
					// triangle placement mode (only allowed on physics-enabled layers)
					canTriangle := false
					if c.Level.LayerMeta != nil && c.CurrentLayer < len(c.Level.LayerMeta) {
						canTriangle = c.Level.LayerMeta[c.CurrentLayer].HasPhysics
					}
					// decide paintValue: if a tileset is loaded and a tile is selected, use that tile index (offset by 3 to avoid colliding with reserved values)
					if c.TilesetImg != nil && c.SelectedTile >= 0 {
						c.PaintValue = c.SelectedTile + 3
						ensureTilesetUsage(c)
						c.Level.TilesetUsage[c.CurrentLayer][gy][gx] = &TilesetEntry{Path: c.TilesetPath, Index: c.SelectedTile, TileW: c.TilesetTileW, TileH: c.TilesetTileH}
					} else if c.TriangleMode && canTriangle {
						if layer[idx] == 2 {
							c.PaintValue = 0
						} else {
							c.PaintValue = 2
						}
						if c.Level.TilesetUsage != nil && c.CurrentLayer < len(c.Level.TilesetUsage) && c.Level.TilesetUsage[c.CurrentLayer] != nil {
							c.Level.TilesetUsage[c.CurrentLayer][gy][gx] = nil
						}
					} else {
						if layer[idx] == 0 {
							c.PaintValue = 1
						} else {
							c.PaintValue = 0
						}
						if c.Level.TilesetUsage != nil && c.CurrentLayer < len(c.Level.TilesetUsage) && c.Level.TilesetUsage[c.CurrentLayer] != nil {
							c.Level.TilesetUsage[c.CurrentLayer][gy][gx] = nil
						}
					}
					// start dragging and apply immediately
					c.Dragging = true
					layer[idx] = c.PaintValue
					c.Level.Layers[c.CurrentLayer] = layer
				}
			}
		}
	} else if pressed && c.PrevMouse && c.Dragging && !c.SpawnMode {
		// dragging: apply paintValue to hovered cell
		if cx, cy, ok := screenToCanvas(mx, my); ok {
			gx := int(math.Floor(cx / float64(c.CellSize)))
			gy := int(math.Floor(cy / float64(c.CellSize)))
			if gx >= 0 && gy >= 0 && gx < c.Level.Width && gy < c.Level.Height {
				idx := gy*c.Level.Width + gx
				if c.Level.Layers == nil || len(c.Level.Layers) == 0 {
					c.Level.Layers = make([][]int, 1)
					c.Level.Layers[0] = make([]int, c.Level.Width*c.Level.Height)
				}
				layer := c.Level.Layers[c.CurrentLayer]
				if layer[idx] != c.PaintValue {
					if c.PendingDeltaMap == nil {
						c.PendingDeltaMap = make(map[int]int)
					}
					if _, seen := c.PendingDeltaMap[idx]; !seen {
						c.PendingDeltaMap[idx] = layer[idx]
					}
					layer[idx] = c.PaintValue
					c.Level.Layers[c.CurrentLayer] = layer
				}
				if c.PaintValue >= 3 {
					ensureTilesetUsage(c)
					c.Level.TilesetUsage[c.CurrentLayer][gy][gx] = &TilesetEntry{Path: c.TilesetPath, Index: c.PaintValue - 3, TileW: c.TilesetTileW, TileH: c.TilesetTileH}
				} else if c.Level.TilesetUsage != nil && c.CurrentLayer < len(c.Level.TilesetUsage) && c.Level.TilesetUsage[c.CurrentLayer] != nil {
					c.Level.TilesetUsage[c.CurrentLayer][gy][gx] = nil
				}
			}
		}
	}

	// end dragging on mouse release
	if !pressed && c.PrevMouse {
		c.Dragging = false
		// if a pending delta was collected during the drag, push it now
		if c.PendingDeltaActive && c.PendingDeltaMap != nil && len(c.PendingDeltaMap) > 0 {
			ld := LayerDelta{Layer: c.PendingDeltaLayer, Changes: c.PendingDeltaMap}
			if c.PushSnapshotDelta != nil {
				c.PushSnapshotDelta(ld)
			} else if c.PushSnapshot != nil {
				// fallback: extract indices and call legacy PushSnapshot
				idxs := make([]int, 0, len(c.PendingDeltaMap))
				for k := range c.PendingDeltaMap {
					idxs = append(idxs, k)
				}
				c.PushSnapshot(c.PendingDeltaLayer, idxs)
			}
		}
		// clear pending delta state
		c.PendingDeltaActive = false
		c.PendingDeltaLayer = 0
		c.PendingDeltaMap = nil
	}
	if !rPressed {
		c.RightDragging = false
	}

	// update prevMouse state (used by tileset panel and click edge detection)
	c.PrevMouse = pressed
}

func ensureTilesetUsage(c *Canvas) {
	if c.Level == nil {
		return
	}
	if c.Level.TilesetUsage == nil || len(c.Level.TilesetUsage) < len(c.Level.Layers) {
		usage := make([][][]*TilesetEntry, len(c.Level.Layers))
		for i := range usage {
			if c.Level.TilesetUsage != nil && i < len(c.Level.TilesetUsage) {
				usage[i] = c.Level.TilesetUsage[i]
			}
		}
		c.Level.TilesetUsage = usage
	}
	if c.CurrentLayer < 0 || c.CurrentLayer >= len(c.Level.TilesetUsage) {
		return
	}
	if c.Level.TilesetUsage[c.CurrentLayer] == nil {
		rows := make([][]*TilesetEntry, c.Level.Height)
		for y := 0; y < c.Level.Height; y++ {
			rows[y] = make([]*TilesetEntry, c.Level.Width)
		}
		c.Level.TilesetUsage[c.CurrentLayer] = rows
	}
}

// Draw renders the canvas contents (background, grid, layers, hover, spawn, controls) into the provided offscreen image.
func (c *Canvas) Draw(canvasImg *ebiten.Image, panelX int) {
	// helper to apply canvas transforms for drawing an image positioned at logical (tx,ty)
	applyCanvas := func(op *ebiten.DrawImageOptions, tx, ty float64) {
		op.GeoM.Translate(tx, ty)                 // position in logical canvas coords
		op.GeoM.Scale(c.CanvasZoom, c.CanvasZoom) // scale canvas
		op.GeoM.Translate(c.CanvasOffsetX, c.CanvasOffsetY)
	}

	// helper to convert screen coords to canvas-local unzoomed coords
	screenToCanvas := func(sx, sy int) (float64, float64, bool) {
		if sx < c.LeftPanelW || sx >= panelX {
			return 0, 0, false
		}
		lx := float64(sx - c.LeftPanelW)
		ly := float64(sy)
		if c.CanvasZoom == 0 {
			c.CanvasZoom = 1.0
		}
		cx := (lx - c.CanvasOffsetX) / c.CanvasZoom
		cy := (ly - c.CanvasOffsetY) / c.CanvasZoom
		return cx, cy, true
	}

	// Draw background layers first (if present)
	if c.Level != nil && len(c.Level.Backgrounds) > 0 {
		if c.Backgrounds != nil {
			c.Backgrounds.Draw(canvasImg, c.CanvasZoom, c.CanvasOffsetX, c.CanvasOffsetY)
		}
	} else if c.Level != nil {
		// Draw base empty grid once
		for y := 0; y < c.Level.Height; y++ {
			for x := 0; x < c.Level.Width; x++ {
				op := &ebiten.DrawImageOptions{}
				applyCanvas(op, float64(x*c.CellSize), float64(y*c.CellSize))
				canvasImg.DrawImage(c.EmptyImg, op)
			}
		}
	}

	if c.Level != nil {
		// Draw layers bottom-to-top (only draw tiles; background already drawn)
		for layerIdx := 0; layerIdx < len(c.Level.Layers); layerIdx++ {
			tileImg := c.LayerTileImgs[layerIdx]

			layer := c.Level.Layers[layerIdx]
			for y := 0; y < c.Level.Height; y++ {
				for x := 0; x < c.Level.Width; x++ {
					idx := y*c.Level.Width + x
					val := layer[idx]
					// solid single-color tile
					if val == 1 {
						op := &ebiten.DrawImageOptions{}
						applyCanvas(op, float64(x*c.CellSize), float64(y*c.CellSize))
						canvasImg.DrawImage(tileImg, op)
					} else if val == 2 {
						// triangle marker
						if c.TriangleImg != nil {
							op := &ebiten.DrawImageOptions{}
							applyCanvas(op, float64(x*c.CellSize), float64(y*c.CellSize))
							canvasImg.DrawImage(c.TriangleImg, op)
						}
					} else if val >= 3 {
						// tileset-based tile (stored as value = index + 3)
						drawn := false
						if c.TilesetImg != nil && c.TilesetTileW > 0 && c.TilesetTileH > 0 {
							tileIndex := val - 3
							cols := c.TilesetImg.Bounds().Dx() / c.TilesetTileW
							rows := c.TilesetImg.Bounds().Dy() / c.TilesetTileH
							if cols > 0 && rows > 0 && tileIndex >= 0 {
								col := tileIndex % cols
								row := tileIndex / cols
								sx := col * c.TilesetTileW
								sy := row * c.TilesetTileH
								if sx >= 0 && sy >= 0 && sx+c.TilesetTileW <= c.TilesetImg.Bounds().Dx() && sy+c.TilesetTileH <= c.TilesetImg.Bounds().Dy() {
									r := image.Rect(sx, sy, sx+c.TilesetTileW, sy+c.TilesetTileH)
									if sub, ok := c.TilesetImg.SubImage(r).(*ebiten.Image); ok {
										op := &ebiten.DrawImageOptions{}
										// tile-scale then canvas transform
										op.GeoM.Scale(float64(c.CellSize)/float64(c.TilesetTileW), float64(c.CellSize)/float64(c.TilesetTileH))
										applyCanvas(op, float64(x*c.CellSize), float64(y*c.CellSize))
										canvasImg.DrawImage(sub, op)
										drawn = true
									}
								}
							}
						}
						if !drawn {
							op := &ebiten.DrawImageOptions{}
							applyCanvas(op, float64(x*c.CellSize), float64(y*c.CellSize))
							if c.MissingImg != nil {
								canvasImg.DrawImage(c.MissingImg, op)
							}
						}
					}

					// optional physics highlight border for physics-enabled layers (draw for any non-empty tile)
					if val != 0 && c.HighlightPhysics && c.Level.LayerMeta != nil && layerIdx < len(c.Level.LayerMeta) && c.Level.LayerMeta[layerIdx].HasPhysics {
						topB := &ebiten.DrawImageOptions{}
						topB.GeoM.Scale(float64(c.CellSize), 1)
						applyCanvas(topB, float64(x*c.CellSize), float64(y*c.CellSize))
						canvasImg.DrawImage(c.BorderImg, topB)
						bottomB := &ebiten.DrawImageOptions{}
						bottomB.GeoM.Scale(float64(c.CellSize), 1)
						applyCanvas(bottomB, float64(x*c.CellSize), float64(y*c.CellSize+c.CellSize-1))
						canvasImg.DrawImage(c.BorderImg, bottomB)
						leftB := &ebiten.DrawImageOptions{}
						leftB.GeoM.Scale(1, float64(c.CellSize))
						applyCanvas(leftB, float64(x*c.CellSize), float64(y*c.CellSize))
						canvasImg.DrawImage(c.BorderImg, leftB)
						rightB := &ebiten.DrawImageOptions{}
						rightB.GeoM.Scale(1, float64(c.CellSize))
						applyCanvas(rightB, float64(x*c.CellSize+c.CellSize-1), float64(y*c.CellSize))
						canvasImg.DrawImage(c.BorderImg, rightB)
					}
				}
			}
		}

		// Hover highlight (draw on top) using canvas transforms
		mx, my := ebiten.CursorPosition()
		if cx, cy, ok := screenToCanvas(mx, my); ok {
			gx := int(math.Floor(cx / float64(c.CellSize)))
			gy := int(math.Floor(cy / float64(c.CellSize)))
			if gx >= 0 && gy >= 0 && gx < c.Level.Width && gy < c.Level.Height {
				hop := &ebiten.DrawImageOptions{}
				applyCanvas(hop, float64(gx*c.CellSize), float64(gy*c.CellSize))
				if c.SpawnMode {
					canvasImg.DrawImage(c.SpawnImgHover, hop)
				} else if c.TriangleMode {
					canvasImg.DrawImage(c.TriangleImgHover, hop)
				} else {
					canvasImg.DrawImage(c.HoverImg, hop)
				}
			}
		}

		// Draw spawn marker: if spawnMode active show at hover cell, else at saved spawn
		if c.SpawnImg != nil {
			sx := c.Level.SpawnX
			sy := c.Level.SpawnY
			if sx >= 0 && sy >= 0 && sx < c.Level.Width && sy < c.Level.Height {
				sop := &ebiten.DrawImageOptions{}
				applyCanvas(sop, float64(sx*c.CellSize), float64(sy*c.CellSize))
				canvasImg.DrawImage(c.SpawnImg, sop)
			}
		}

		// Controls text (draw inside canvas so it is clipped to canvas area)
		c.ControlsText.Draw(canvasImg, c)
	}
}
