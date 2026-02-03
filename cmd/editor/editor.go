package main

import (
	"bytes"
	"encoding/json"
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
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/assets"
	"golang.org/x/image/colornames"
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
	// save prompt UI (now handled by modal Prompt)
	// new level size prompt UI
	newLevelPrompt bool
	newLevelInput  string
	newLevelStage  int // 0=width, 1=height
	newLevelWidth  int
	newLevelError  string
	currentLayer   int
	prevCyclePrev  bool
	prevCycleNext  bool
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

	// transition placement mode: place rectangular zones that point to another file
	transitionMode     bool
	pendingTransition  *Transition
	transitionFillImg  *ebiten.Image
	transitionDragging bool
	transitionStartX   int
	transitionStartY   int

	highlightPhysics bool
	borderImg        *ebiten.Image
	// tileset panel component
	tilesetPanel *TilesetPanel
	entityPanel  *EntityPanel
	// entities currently placed in the scene (simple list of entity filenames)
	sceneEntities []string
	// placement state for dragging an entity from the scene list onto the canvas
	placingActive          bool
	placingIndex           int
	placingImg             *ebiten.Image
	placingIgnoreNextClick bool
	entitySpriteCache      map[string]*ebiten.Image // cache loaded entity sprite images by sprite path
	// controls text component
	controlsText ControlsText

	// modal prompt component (captures input when open)
	prompt *Prompt

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

	// yellow semi-transparent 1x1 used to draw transition rectangles
	tf := ebiten.NewImage(1, 1)
	tf.Fill(color.RGBA{R: 0xff, G: 0xff, B: 0x00, A: 0x88})
	eg.transitionFillImg = tf

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

	// prompt component
	eg.prompt = NewPrompt()
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
	meta[0] = LayerMeta{HasPhysics: false, Color: "#3c78ff", Name: "Layer 0"}
	g.level = &Level{Width: w, Height: h, Layers: layers, LayerMeta: meta}
	g.currentLayer = 0
	// setup per-layer images
	g.layerTileImgs = make([]*ebiten.Image, len(g.level.LayerMeta))
	for i := range g.level.LayerMeta {
		g.layerTileImgs[i] = layerImageFromHex(g.cellSize, g.level.LayerMeta[i].Color)
	}
}

// StartNewLevelPrompt shows a prompt for width/height before creating a new level.
func (g *Editor) StartNewLevelPrompt() {
	g.newLevelPrompt = true
	g.newLevelInput = ""
	g.newLevelStage = 0
	g.newLevelWidth = 0
	g.newLevelError = ""
	g.filename = ""
	g.level = nil
}

// Update handles input and editor state changes.
func (g *Editor) Update() error {
	// If a modal prompt is open, let it handle input and ignore other keys.
	if g.prompt != nil && g.prompt.IsOpen() {
		g.prompt.Update()
		return nil
	}

	if g.newLevelPrompt {
		for _, r := range ebiten.InputChars() {
			if r >= '0' && r <= '9' {
				g.newLevelInput += string(r)
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if len(g.newLevelInput) > 0 {
				g.newLevelInput = g.newLevelInput[:len(g.newLevelInput)-1]
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if g.newLevelInput != "" {
				val, err := strconv.Atoi(g.newLevelInput)
				if err != nil || val <= 0 {
					g.newLevelError = "Enter a positive integer"
					g.newLevelInput = ""
				} else if g.newLevelStage == 0 {
					g.newLevelWidth = val
					g.newLevelInput = ""
					g.newLevelStage = 1
					g.newLevelError = ""
				} else {
					g.Init(g.newLevelWidth, val)
					g.newLevelPrompt = false
					g.newLevelStage = 0
					g.newLevelInput = ""
					g.newLevelWidth = 0
					g.newLevelError = ""
				}
			}
		}
		return nil
	}
	if g.level == nil {
		return nil
	}

	// Update left-side panels (handle input there)
	if g.entityPanel != nil {
		g.entityPanel.Update(g)
	}

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

	// Toggle transition placement mode (G)
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		g.transitionMode = !g.transitionMode
	}

	// Toggle fill mode (F)
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if g.canvas != nil {
			g.canvas.FillMode = !g.canvas.FillMode
		}
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

	// Picking up or deleting placed entities: left-click picks up (move), right-click deletes
	if !g.placingActive && g.level != nil {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			if cx, cy, ok := screenToCanvas(mx, my); ok {
				gx := int(math.Floor(cx / float64(g.cellSize)))
				gy := int(math.Floor(cy / float64(g.cellSize)))
				if gx >= 0 && gy >= 0 && gx < g.level.Width && gy < g.level.Height {
					// find the top-most placed entity at this cell (search from end)
					for ei := len(g.level.Entities) - 1; ei >= 0; ei-- {
						pe := g.level.Entities[ei]
						if pe.X == gx && pe.Y == gy {
							log.Printf("EntityPanel: clicked entity at idx=%d name=%s sprite=%s", ei, pe.Name, pe.Sprite)
							// right-click => delete
							if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
								name := pe.Name
								// remove the instance
								g.level.Entities = append(g.level.Entities[:ei], g.level.Entities[ei+1:]...)
								log.Printf("EntityPanel: deleted entity %s at (%d,%d)", pe.Name, gx, gy)
								// if no other placed instances of this entity remain, remove it from the scene palette
								still := false
								for _, rest := range g.level.Entities {
									if rest.Name == name {
										still = true
										break
									}
								}
								if !still {
									for si := len(g.sceneEntities) - 1; si >= 0; si-- {
										if g.sceneEntities[si] == name {
											g.sceneEntities = append(g.sceneEntities[:si], g.sceneEntities[si+1:]...)
											log.Printf("EntityPanel: removed %s from sceneEntities", name)
											break
										}
									}
								}
								break
							}
							// left-click => pick up for moving
							if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
								log.Printf("EntityPanel: picking up entity %s at idx=%d", pe.Name, ei)
								name := pe.Name
								// ensure present in scene palette
								found := false
								for i := range g.sceneEntities {
									if g.sceneEntities[i] == name {
										g.placingIndex = i
										found = true
										break
									}
								}
								if !found {
									g.sceneEntities = append(g.sceneEntities, name)
									g.placingIndex = len(g.sceneEntities) - 1
								}
								// load sprite into cache and set preview
								if g.entitySpriteCache == nil {
									g.entitySpriteCache = make(map[string]*ebiten.Image)
								}
								if img, ok := g.entitySpriteCache[pe.Sprite]; ok {
									g.placingImg = img
								} else if pe.Sprite != "" {
									if img2, err := assets.LoadImage(pe.Sprite); err == nil {
										g.entitySpriteCache[pe.Sprite] = img2
										g.placingImg = img2
									} else {
										tried := []string{pe.Sprite, filepath.Join("assets", pe.Sprite), filepath.Base(pe.Sprite)}
										var fallback *ebiten.Image
										for _, p := range tried {
											if b, e := os.ReadFile(p); e == nil {
												if im, _, e2 := image.Decode(bytes.NewReader(b)); e2 == nil {
													fallback = ebiten.NewImageFromImage(im)
													break
												}
											}
										}
										if fallback != nil {
											g.entitySpriteCache[pe.Sprite] = fallback
											g.placingImg = fallback
										}
									}
								}
								if g.placingImg == nil {
									g.placingImg = g.missingImg
								}
								// remove the entity instance from level so it's being moved
								g.level.Entities = append(g.level.Entities[:ei], g.level.Entities[ei+1:]...)
								g.placingActive = true
								g.placingIgnoreNextClick = true
								break
							}
						}
					}
				}
			}
		}
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

	// Transition placement mode: support drag-to-create and right-click deletion.
	if g.transitionMode {
		// Right-click deletes a transition under the cursor
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
			if cx, cy, ok := screenToCanvas(mx, my); ok && g.level != nil {
				gx := int(math.Floor(cx / float64(g.cellSize)))
				gy := int(math.Floor(cy / float64(g.cellSize)))
				for i := range g.level.Transitions {
					tr := g.level.Transitions[i]
					if gx >= tr.X && gx < tr.X+tr.W && gy >= tr.Y && gy < tr.Y+tr.H {
						// remove transition
						g.level.Transitions = append(g.level.Transitions[:i], g.level.Transitions[i+1:]...)
						break
					}
				}
			}
		}

		// Start drag on left-button just pressed
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if cx, cy, ok := screenToCanvas(mx, my); ok && g.level != nil {
				gx := int(math.Floor(cx / float64(g.cellSize)))
				gy := int(math.Floor(cy / float64(g.cellSize)))
				if gx >= 0 && gy >= 0 && gx < g.level.Width && gy < g.level.Height {
					g.transitionStartX = gx
					g.transitionStartY = gy
					g.transitionDragging = true
					g.pendingTransition = &Transition{X: gx, Y: gy, W: 1, H: 1, Target: ""}
				}
			}
		}

		// Update pending rect while dragging
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.transitionDragging {
			if cx, cy, ok := screenToCanvas(mx, my); ok && g.level != nil {
				gx := int(math.Floor(cx / float64(g.cellSize)))
				gy := int(math.Floor(cy / float64(g.cellSize)))
				if gx < 0 {
					gx = 0
				}
				if gy < 0 {
					gy = 0
				}
				if gx >= g.level.Width {
					gx = g.level.Width - 1
				}
				if gy >= g.level.Height {
					gy = g.level.Height - 1
				}
				x0 := g.transitionStartX
				y0 := g.transitionStartY
				minX := x0
				maxX := gx
				if gx < x0 {
					minX = gx
					maxX = x0
				}
				minY := y0
				maxY := gy
				if gy < y0 {
					minY = gy
					maxY = y0
				}
				if g.pendingTransition != nil {
					g.pendingTransition.X = minX
					g.pendingTransition.Y = minY
					g.pendingTransition.W = maxX - minX + 1
					g.pendingTransition.H = maxY - minY + 1
				} else {
					g.pendingTransition = &Transition{X: minX, Y: minY, W: maxX - minX + 1, H: maxY - minY + 1}
				}
			}
		}

		// Finish drag on left-button release: open prompt to name target then persist
		if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) && g.transitionDragging {
			g.transitionDragging = false
			if g.pendingTransition != nil && g.level != nil {
				// finalize pending transition: prompt for ID, then target, then linked ID
				if g.prompt != nil {
					// capture a copy of the pending transition now (so callbacks can use it after we clear g.pendingTransition)
					tr := *g.pendingTransition
					// first prompt: transition ID
					g.prompt.Open("Transition ID (press Enter to confirm, Esc to cancel):", "", func(id string) {
						if id == "" {
							return
						}
						tr.ID = id

						// second prompt: target level filename
						g.prompt.Open("Transition target (press Enter to confirm, Esc to cancel):", "", func(name string) {
							if name == "" {
								return
							}
							if filepath.Ext(name) == "" {
								name = name + ".json"
							}
							// ensure the transition does not leave physics tiles inside its area
							if g.level != nil {
								// clear tiles (and tileset usage) for any physics-enabled layers
								if g.level.LayerMeta != nil {
									for li := range g.level.Layers {
										if li < len(g.level.LayerMeta) && g.level.LayerMeta[li].HasPhysics {
											layer := g.level.Layers[li]
											for yy := tr.Y; yy < tr.Y+tr.H; yy++ {
												if yy < 0 || yy >= g.level.Height {
													continue
												}
												for xx := tr.X; xx < tr.X+tr.W; xx++ {
													if xx < 0 || xx >= g.level.Width {
														continue
													}
													idx := yy*g.level.Width + xx
													if idx >= 0 && idx < len(layer) {
														layer[idx] = 0
													}
													if g.level.TilesetUsage != nil && li < len(g.level.TilesetUsage) {
														if g.level.TilesetUsage[li] != nil && yy < len(g.level.TilesetUsage[li]) && xx < len(g.level.TilesetUsage[li][yy]) {
															g.level.TilesetUsage[li][yy][xx] = nil
														}
													}
												}
											}
											g.level.Layers[li] = layer
										}
									}
								}
							}

							tr.Target = name

							// third prompt: linked transition ID
							g.prompt.Open("Linked transition ID (press Enter to confirm, Esc to cancel):", "", func(link string) {
								tr.LinkID = link
								g.level.Transitions = append(g.level.Transitions, tr)
							})
						})
					})
				}
				// clear pending regardless — we captured a copy for the prompt callbacks
				g.pendingTransition = nil
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

		g.canvas.Update(mx, my, panelX, inTilesetPanel, g.placingActive)

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
		name := fmt.Sprintf("Layer %d", len(g.level.LayerMeta))
		g.level.LayerMeta = append(g.level.LayerMeta, LayerMeta{HasPhysics: false, Color: "#3c78ff", Name: name})
		// create image for new layer
		g.layerTileImgs = append(g.layerTileImgs, layerImageFromHex(g.cellSize, "#3c78ff"))
		g.currentLayer = len(g.level.Layers) - 1
	}

	// Toggle physics flag for current layer (H)
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		if g.level.LayerMeta == nil || g.currentLayer >= len(g.level.LayerMeta) {
			// ensure meta exists
			for len(g.level.LayerMeta) <= g.currentLayer {
				name := fmt.Sprintf("Layer %d", len(g.level.LayerMeta))
				g.level.LayerMeta = append(g.level.LayerMeta, LayerMeta{HasPhysics: false, Color: "#3c78ff", Name: name})
				g.layerTileImgs = append(g.layerTileImgs, layerImageFromHex(g.cellSize, "#3c78ff"))
			}
		}
		g.level.LayerMeta[g.currentLayer].HasPhysics = !g.level.LayerMeta[g.currentLayer].HasPhysics
	}

	// Set per-layer parallax (L) — opens modal prompt to enter a float value.
	if inpututil.IsKeyJustPressed(ebiten.KeyL) && !(ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)) {
		if g.level == nil {
			// nothing to do
		} else {
			// ensure meta exists for current layer
			g.ensureLayerMetaLen(len(g.level.Layers))
			cur := ""
			if g.currentLayer < len(g.level.LayerMeta) {
				if g.level.LayerMeta[g.currentLayer].Parallax != 0 {
					cur = fmt.Sprintf("%v", g.level.LayerMeta[g.currentLayer].Parallax)
				}

				// (Ctrl+L handled separately)
			}

			// Ctrl+L: toggle line-draw tool
			if inpututil.IsKeyJustPressed(ebiten.KeyL) && (ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)) {
				if g.canvas != nil {
					g.canvas.LineMode = !g.canvas.LineMode
					log.Printf("Line tool: %v", g.canvas.LineMode)
				}
			}
			if g.prompt != nil {
				ii := g.currentLayer
				g.prompt.Open("Parallax (float):", cur, func(s string) {
					if s == "" {
						// empty => reset to 0 (omitempty will drop it on save)
						g.ensureLayerMetaLen(len(g.level.Layers))
						if ii < len(g.level.LayerMeta) {
							g.level.LayerMeta[ii].Parallax = 0
						}
						return
					}
					f, err := strconv.ParseFloat(s, 64)
					if err != nil {
						log.Printf("failed to parse parallax '%s': %v", s, err)
						return
					}
					g.ensureLayerMetaLen(len(g.level.Layers))
					if ii < len(g.level.LayerMeta) {
						g.level.LayerMeta[ii].Parallax = f
					}
				})
			}
		}
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

	// Save if S pressed: show filename prompt when saving a new (unsaved) level
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if g.filename == "" {
			// open modal prompt to collect filename
			g.prompt.Open("Save as (press Enter to confirm, Esc to cancel):", "", func(name string) {
				if name == "" {
					return
				}
				if err := os.MkdirAll("levels", 0755); err != nil {
					log.Printf("save error: %v", err)
					return
				}
				if filepath.Ext(name) == "" {
					name = name + ".json"
				}
				g.filename = filepath.Join("levels", name)
				if err := g.Save(); err != nil {
					log.Printf("save error: %v", err)
				} else {
					log.Printf("saved to %s", g.filename)
				}
			})
		} else {
			if err := g.Save(); err != nil {
				log.Printf("save error: %v", err)
			} else {
				log.Printf("saved to %s", g.filename)
			}
		}
	}

	// (transition prompt is handled by the modal Prompt when opened)

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

	// Handle entity placement commit / cancel
	if g.placingActive {
		// cancel placement with right-click or Esc
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.placingActive = false
			g.placingImg = nil
			g.placingIgnoreNextClick = false
		} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			// ignore the immediate left-click that started a pickup
			if g.placingIgnoreNextClick {
				g.placingIgnoreNextClick = false
			} else {
				// commit placement if over canvas
				if cx, cy, ok := screenToCanvas(mx, my); ok && g.level != nil {
					gx := int(math.Floor(cx / float64(g.cellSize)))
					gy := int(math.Floor(cy / float64(g.cellSize)))
					if gx >= 0 && gy >= 0 && gx < g.level.Width && gy < g.level.Height {
						name := ""
						if g.placingIndex >= 0 && g.placingIndex < len(g.sceneEntities) {
							name = g.sceneEntities[g.placingIndex]
						}
						sprite := ""
						if name != "" {
							if b, err := os.ReadFile(filepath.Join("entities", name)); err == nil {
								var ent struct {
									Name   string `json:"name"`
									Sprite string `json:"sprite"`
								}
								if json.Unmarshal(b, &ent) == nil {
									sprite = ent.Sprite
								}
							}
							// append placed entity to level
							pe := PlacedEntity{Name: name, Sprite: sprite, X: gx, Y: gy}
							g.level.Entities = append(g.level.Entities, pe)
						}
					}
				}
				g.placingActive = false
				g.placingImg = nil
			}
		}
	}

	return nil
}

// Draw renders the editor.
func (g *Editor) Draw(screen *ebiten.Image) {
	screen.Clear()
	screen.Fill(colornames.Darkslategrey)

	if g.newLevelPrompt || g.level == nil {
		screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0xff})
		screenW := screen.Bounds().Dx()
		screenH := screen.Bounds().Dy()
		// semi-transparent backdrop across screen
		o := &ebiten.DrawImageOptions{}
		back := ebiten.NewImage(screenW, 64)
		back.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0x88})
		o.GeoM.Translate(0, float64(screenH/2-32))
		screen.DrawImage(back, o)
		prompt := ""
		if g.newLevelStage == 0 {
			prompt = fmt.Sprintf("New level width (cells): %s", g.newLevelInput)
		} else {
			prompt = fmt.Sprintf("New level height (cells): %s", g.newLevelInput)
		}
		if g.newLevelError != "" {
			prompt = fmt.Sprintf("%s\n%s", prompt, g.newLevelError)
		}
		ebitenutil.DebugPrintAt(screen, prompt, 16, screenH/2-16)
		return
	}

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

	// Draw transition rectangles (yellow semi-transparent) onto the canvas image
	if g.level != nil && g.transitionFillImg != nil {
		for _, tr := range g.level.Transitions {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(tr.W*g.cellSize), float64(tr.H*g.cellSize))
			applyCanvas(op, float64(tr.X*g.cellSize), float64(tr.Y*g.cellSize))
			g.canvasImg.DrawImage(g.transitionFillImg, op)
		}
		if g.pendingTransition != nil {
			tr := *g.pendingTransition
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(float64(tr.W*g.cellSize), float64(tr.H*g.cellSize))
			applyCanvas(op, float64(tr.X*g.cellSize), float64(tr.Y*g.cellSize))
			g.canvasImg.DrawImage(g.transitionFillImg, op)
		}
	}

	// Draw placed entity instances onto the canvas (cache+load sprites as needed)
	if g.level != nil && len(g.level.Entities) > 0 {
		for _, pe := range g.level.Entities {
			var img *ebiten.Image
			if g.entitySpriteCache != nil {
				img = g.entitySpriteCache[pe.Sprite]
			}
			if img == nil {
				if pe.Sprite != "" {
					if img2, err := assets.LoadImage(pe.Sprite); err == nil {
						if g.entitySpriteCache == nil {
							g.entitySpriteCache = make(map[string]*ebiten.Image)
						}
						g.entitySpriteCache[pe.Sprite] = img2
						img = img2
					} else {
						// try filesystem fallbacks
						tried := []string{pe.Sprite, filepath.Join("assets", pe.Sprite), filepath.Base(pe.Sprite)}
						for _, p := range tried {
							if b, e := os.ReadFile(p); e == nil {
								if im, _, e2 := image.Decode(bytes.NewReader(b)); e2 == nil {
									img = ebiten.NewImageFromImage(im)
									if g.entitySpriteCache == nil {
										g.entitySpriteCache = make(map[string]*ebiten.Image)
									}
									g.entitySpriteCache[pe.Sprite] = img
									break
								}
							}
						}
					}
				}
			}
			if img == nil {
				img = g.missingImg
			}
			if img != nil {
				// scale sprite to fit cell size
				iw := img.Bounds().Dx()
				ih := img.Bounds().Dy()
				if iw > 0 && ih > 0 {
					op := &ebiten.DrawImageOptions{}
					op.GeoM.Scale(float64(g.cellSize)/float64(iw), float64(g.cellSize)/float64(ih))
					applyCanvas(op, float64(pe.X*g.cellSize), float64(pe.Y*g.cellSize))
					g.canvasImg.DrawImage(img, op)
				}
			}
		}
	}

	// If placingActive with a valid preview image, dim the canvas to make the preview stand out
	if g.placingActive && g.placingImg != nil {
		if dark := ebiten.NewImage(g.canvasImg.Bounds().Dx(), g.canvasImg.Bounds().Dy()); dark != nil {
			dark.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0x66})
			dOp := &ebiten.DrawImageOptions{}
			g.canvasImg.DrawImage(dark, dOp)
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

	// Draw placement preview on top of hover if active
	if g.placingActive && g.placingImg != nil {
		if cx, cy, ok := g.screenToCanvasPoint(mx, my, panelX); ok {
			gx := int(math.Floor(cx / float64(g.cellSize)))
			gy := int(math.Floor(cy / float64(g.cellSize)))
			if gx >= 0 && gy >= 0 && gx < g.level.Width && gy < g.level.Height {
				drawOp := &ebiten.DrawImageOptions{}
				w := g.placingImg.Bounds().Dx()
				h := g.placingImg.Bounds().Dy()
				if w > 0 && h > 0 {
					drawOp.GeoM.Scale(float64(g.cellSize)/float64(w), float64(g.cellSize)/float64(h))
				}
				applyCanvas(drawOp, float64(gx*g.cellSize), float64(gy*g.cellSize))
				g.canvasImg.DrawImage(g.placingImg, drawOp)
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
	g.entityPanel.Draw(screen, g)

	// Draw modal prompt overlay if active
	if g.prompt != nil {
		g.prompt.Draw(screen)
	}
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
	tool := "Brush"
	if c.LineMode {
		tool = "Line"
	} else if c.FillMode {
		tool = "Fill"
	}

	instr := fmt.Sprintf("Left-click: toggle tile   F: fill   S: save   Q/E: cycle layers   N: new layer   H: toggle physics   Y: highlight physics   P: place spawn   T: triangle mode  Y: highlight physics  B: add background   File: %s\nTool=%s  W=%d H=%d Cell=%d Layer=%d has_physics=%v color=%s spawn=(%d,%d) spawnMode=%v triangleMode=%v backgrounds=%d",
		filename, tool, func() int {
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

// MoveLayerUp moves layer at index idx one position up (increasing its index).
func (g *Editor) MoveLayerUp(idx int) {
	if g.level == nil || idx < 0 || idx >= len(g.level.Layers)-1 {
		return
	}
	// ensure metadata and tile images align with layers so names move with layers
	g.ensureLayerMetaLen(len(g.level.Layers))
	// swap layers and metadata and pre-rendered images
	g.level.Layers[idx], g.level.Layers[idx+1] = g.level.Layers[idx+1], g.level.Layers[idx]
	g.level.LayerMeta[idx], g.level.LayerMeta[idx+1] = g.level.LayerMeta[idx+1], g.level.LayerMeta[idx]
	if len(g.layerTileImgs) == len(g.level.LayerMeta) {
		g.layerTileImgs[idx], g.layerTileImgs[idx+1] = g.layerTileImgs[idx+1], g.layerTileImgs[idx]
	}
	// adjust currentLayer if it was one of the swapped
	if g.currentLayer == idx {
		g.currentLayer = idx + 1
	} else if g.currentLayer == idx+1 {
		g.currentLayer = idx
	}
}

// MoveLayerDown moves layer at index idx one position down (decreasing its index).
func (g *Editor) MoveLayerDown(idx int) {
	if g.level == nil || idx <= 0 || idx >= len(g.level.Layers) {
		return
	}
	// ensure metadata and tile images align with layers so names move with layers
	g.ensureLayerMetaLen(len(g.level.Layers))
	g.level.Layers[idx], g.level.Layers[idx-1] = g.level.Layers[idx-1], g.level.Layers[idx]
	g.level.LayerMeta[idx], g.level.LayerMeta[idx-1] = g.level.LayerMeta[idx-1], g.level.LayerMeta[idx]
	if len(g.layerTileImgs) == len(g.level.LayerMeta) {
		g.layerTileImgs[idx], g.layerTileImgs[idx-1] = g.layerTileImgs[idx-1], g.layerTileImgs[idx]
	}
	if g.currentLayer == idx {
		g.currentLayer = idx - 1
	} else if g.currentLayer == idx-1 {
		g.currentLayer = idx
	}
}

// SelectLayer selects the given layer index as the current editing layer.
func (g *Editor) SelectLayer(idx int) {
	if g.level == nil || idx < 0 || idx >= len(g.level.Layers) {
		return
	}
	g.currentLayer = idx
}

// ensureLayerMetaLen extends LayerMeta and layerTileImgs so they match the
// provided number of layers. This ensures names and pre-rendered images
// move together when layers are reordered.
func (g *Editor) ensureLayerMetaLen(n int) {
	if g.level == nil {
		return
	}
	for len(g.level.LayerMeta) < n {
		name := fmt.Sprintf("Layer %d", len(g.level.LayerMeta))
		g.level.LayerMeta = append(g.level.LayerMeta, LayerMeta{HasPhysics: false, Color: "#3c78ff", Name: name})
		g.layerTileImgs = append(g.layerTileImgs, layerImageFromHex(g.cellSize, "#3c78ff"))
	}
}
