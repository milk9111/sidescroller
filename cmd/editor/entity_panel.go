package main

import (
	"bytes"
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
	"github.com/milk9111/sidescroller/assets"
)

const (
	leftPanelWidth = 200
)

type EntityPanel struct {
	panelBgImg *ebiten.Image
	// double-click state
	lastClickTime time.Time
	lastClickIdx  int
	// cached available entities (filenames)
	available []string
}

func NewEntityPanel() *EntityPanel {
	bg := ebiten.NewImage(1, 1)
	bg.Fill(color.RGBA{0x0b, 0x14, 0x2a, 0xff}) // dark blue

	ep := &EntityPanel{
		panelBgImg:   bg,
		lastClickIdx: -1,
	}
	// try to read available entities from the entities/ folder
	if entries, err := os.ReadDir("entities"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			// only include .json files
			if filepath.Ext(name) == ".json" {
				ep.available = append(ep.available, name)
			}
		}
	} else {
		log.Printf("EntityPanel: failed to read entities/: %v", err)
	}

	return ep
}

// Update handles input for the layers panel. Called from Editor.Update.
func (ep *EntityPanel) Update(g *Editor) {
	if g == nil || g.level == nil {
		return
	}
	mx, my := ebiten.CursorPosition()
	// only handle clicks that occur inside the left panel region
	if mx < 0 || mx >= leftPanelWidth {
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		listX := 8
		listY := 28
		itemH := 20

		// Available entities list: left-click to start placement (also add to scene palette)
		for idx, name := range ep.available {
			y := listY + idx*itemH
			if mx >= listX && mx <= leftPanelWidth-8 && my >= y && my <= y+itemH {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					// ensure present in scene palette
					found := false
					for i := range g.sceneEntities {
						if g.sceneEntities[i] == name {
							found = true
							g.placingIndex = i
							break
						}
					}
					if !found {
						g.sceneEntities = append(g.sceneEntities, name)
						g.placingIndex = len(g.sceneEntities) - 1
					}
					// read entity JSON and load sprite for placement preview
					b, err := os.ReadFile(filepath.Join("entities", name))
					if err == nil {
						var ent struct {
							Name   string `json:"name"`
							Sprite string `json:"sprite"`
						}
						if json.Unmarshal(b, &ent) == nil {
							if g.entitySpriteCache == nil {
								g.entitySpriteCache = make(map[string]*ebiten.Image)
							}
							if img, ok := g.entitySpriteCache[ent.Sprite]; ok {
								g.placingImg = img
								log.Printf("EntityPanel: using cached sprite %s for %s", ent.Sprite, name)
							} else {
								if img2, err := assets.LoadImage(ent.Sprite); err == nil {
									g.entitySpriteCache[ent.Sprite] = img2
									g.placingImg = img2
									log.Printf("EntityPanel: loaded sprite %s for %s", ent.Sprite, name)
								} else {
									// try filesystem fallbacks: direct path, assets/<path>, basename
									tried := []string{ent.Sprite, filepath.Join("assets", ent.Sprite), filepath.Base(ent.Sprite)}
									var fallbackImg *ebiten.Image
									for _, p := range tried {
										if b2, e2 := os.ReadFile(p); e2 == nil {
											if im, _, e3 := image.Decode(bytes.NewReader(b2)); e3 == nil {
												fallbackImg = ebiten.NewImageFromImage(im)
												log.Printf("EntityPanel: loaded sprite from fs %s for %s", p, name)
												break
											}
										}
									}
									if fallbackImg != nil {
										g.entitySpriteCache[ent.Sprite] = fallbackImg
										g.placingImg = fallbackImg
									} else {
										log.Printf("EntityPanel: failed to load sprite %s for %s: %v", ent.Sprite, name, err)
									}
								}
							}
							// fallback to missing image so placement doesn't blank the map
							if g.placingImg == nil {
								g.placingImg = g.missingImg
							}
						}
					}
					g.placingActive = true
					g.placingIgnoreNextClick = true
				}
				return
			}
		}

		// Scene entities list: left-click to start placing an instance, right-click to remove from palette
		sceneStart := listY + len(ep.available)*itemH + 12
		for idx := range g.sceneEntities {
			y := sceneStart + idx*itemH
			if mx >= listX && mx <= leftPanelWidth-8 && my >= y && my <= y+itemH {
				if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
					// remove from palette
					log.Printf("EntityPanel: remove scene entity idx=%d", idx)
					g.sceneEntities = append(g.sceneEntities[:idx], g.sceneEntities[idx+1:]...)
				} else if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
					// start placement of this type
					g.placingIndex = idx
					name := g.sceneEntities[idx]
					b, err := os.ReadFile(filepath.Join("entities", name))
					if err == nil {
						var ent struct {
							Name   string `json:"name"`
							Sprite string `json:"sprite"`
						}
						if json.Unmarshal(b, &ent) == nil {
							if g.entitySpriteCache == nil {
								g.entitySpriteCache = make(map[string]*ebiten.Image)
							}
							if img, ok := g.entitySpriteCache[ent.Sprite]; ok {
								g.placingImg = img
								log.Printf("EntityPanel: using cached sprite %s for scene %s", ent.Sprite, name)
							} else {
								if img2, err := assets.LoadImage(ent.Sprite); err == nil {
									g.entitySpriteCache[ent.Sprite] = img2
									g.placingImg = img2
									log.Printf("EntityPanel: loaded sprite %s for scene %s", ent.Sprite, name)
								} else {
									tried := []string{ent.Sprite, filepath.Join("assets", ent.Sprite), filepath.Base(ent.Sprite)}
									var fallbackImg *ebiten.Image
									for _, p := range tried {
										if b2, e2 := os.ReadFile(p); e2 == nil {
											if im, _, e3 := image.Decode(bytes.NewReader(b2)); e3 == nil {
												fallbackImg = ebiten.NewImageFromImage(im)
												log.Printf("EntityPanel: loaded sprite from fs %s for scene %s", p, name)
												break
											}
										}
									}
									if fallbackImg != nil {
										g.entitySpriteCache[ent.Sprite] = fallbackImg
										g.placingImg = fallbackImg
									} else {
										log.Printf("EntityPanel: failed to load sprite %s for scene %s: %v", ent.Sprite, name, err)
									}
								}
							}
							if g.placingImg == nil {
								g.placingImg = g.missingImg
							}
						}
					}
					g.placingActive = true
					g.placingIgnoreNextClick = true
				}
				log.Printf("EntityPanel: selected scene entity idx=%d name=%s", idx, g.sceneEntities[idx])
				return
			}
		}

		// Layers list (below scene entities) - preserve original behavior
		layerStart := sceneStart + len(g.sceneEntities)*itemH + 12
		layerItemH := 28
		// iterate visually from top to bottom so clicks match display
		n := len(g.level.Layers)
		displayOrder := make([]int, 0, n)
		for i := n - 1; i >= 0; i-- {
			displayOrder = append(displayOrder, i)
		}
		for pos, idx := range displayOrder {
			y := layerStart + pos*layerItemH
			btnW := 24
			btnX := leftPanelWidth - btnW - 8
			upY := y
			downY := y + 12
			// up
			if mx >= btnX && mx <= btnX+btnW && my >= upY && my <= upY+12 {
				log.Printf("EntityPanel: up click at mx=%d my=%d idx=%d btnX=%d upY=%d", mx, my, idx, btnX, upY)
				g.MoveLayerUp(idx)
				return
			}
			// down
			if mx >= btnX && mx <= btnX+btnW && my >= downY && my <= downY+12 {
				log.Printf("EntityPanel: down click at mx=%d my=%d idx=%d btnX=%d downY=%d", mx, my, idx, btnX, downY)
				g.MoveLayerDown(idx)
				return
			}
			// select (single-click selects; double-click renames)
			if mx >= listX && mx <= leftPanelWidth-40 && my >= y && my <= y+layerItemH {
				now := time.Now()
				double := false
				if ep.lastClickIdx == idx && now.Sub(ep.lastClickTime) <= 400*time.Millisecond {
					double = true
				}
				ep.lastClickTime = now
				ep.lastClickIdx = idx
				if double {
					// open rename prompt for this layer
					// ensure LayerMeta exists
					g.ensureLayerMetaLen(len(g.level.Layers))
					current := ""
					if idx < len(g.level.LayerMeta) {
						current = g.level.LayerMeta[idx].Name
					}
					if current == "" {
						current = fmt.Sprintf("Layer %d", idx)
					}
					log.Printf("EntityPanel: rename double-click idx=%d current=%s", idx, current)
					if g.prompt != nil {
						// capture idx for closure
						ii := idx
						g.prompt.Open("Rename layer:", current, func(s string) {
							if s == "" {
								return
							}
							g.ensureLayerMetaLen(len(g.level.Layers))
							if ii < len(g.level.LayerMeta) {
								g.level.LayerMeta[ii].Name = s
							}
						})
					}
					return
				}
				// single-click selects
				log.Printf("EntityPanel: select click at mx=%d my=%d idx=%d y=%d", mx, my, idx, y)
				g.SelectLayer(idx)
				return
			}
		}
	}
}

// Draw renders the layers panel; input is handled in Update.
func (ep *EntityPanel) Draw(screen *ebiten.Image, g *Editor) {
	// background
	lpOp := &ebiten.DrawImageOptions{}
	lpOp.GeoM.Scale(float64(leftPanelWidth), float64(screen.Bounds().Dy()))
	lpOp.GeoM.Translate(0, 0)
	screen.DrawImage(ep.panelBgImg, lpOp)
	ebitenutil.DebugPrintAt(screen, "Available Entities:", 8, 8)

	if g == nil || g.level == nil {
		return
	}

	listX := 8
	listY := 28
	itemH := 20

	// draw available entities
	for i, name := range ep.available {
		y := listY + i*itemH
		display := name
		// trim extension for display
		if ext := filepath.Ext(name); ext != "" {
			display = name[:len(name)-len(ext)]
		}
		ebitenutil.DebugPrintAt(screen, display, listX, y)
	}

	// scene entities
	sceneStart := listY + len(ep.available)*itemH + 12
	ebitenutil.DebugPrintAt(screen, "Scene Entities:", listX, sceneStart-16)
	for i, name := range g.sceneEntities {
		y := sceneStart + i*itemH
		display := name
		if ext := filepath.Ext(name); ext != "" {
			display = name[:len(name)-len(ext)]
		}
		ebitenutil.DebugPrintAt(screen, display, listX, y)
	}

	// layers list (below scene entities)
	layerStart := sceneStart + len(g.sceneEntities)*itemH + 12
	ebitenutil.DebugPrintAt(screen, "Layers:", listX, layerStart-16)
	// draw layers top-first, showing visual position numbers (1., 2., ...)
	n := len(g.level.Layers)
	displayOrder := make([]int, 0, n)
	for i := n - 1; i >= 0; i-- {
		displayOrder = append(displayOrder, i)
	}
	layerItemH := 28
	for pos, idx := range displayOrder {
		y := layerStart + pos*layerItemH
		name := ""
		if idx < len(g.level.LayerMeta) {
			name = g.level.LayerMeta[idx].Name
		}
		if name == "" {
			name = fmt.Sprintf("Layer %d", idx)
		}
		label := fmt.Sprintf("%d. %s", pos+1, name)
		if idx == g.currentLayer {
			ebitenutil.DebugPrintAt(screen, "> "+label, listX, y)
		} else {
			ebitenutil.DebugPrintAt(screen, label, listX, y)
		}

		// up/down indicators
		btnW := 24
		btnX := leftPanelWidth - btnW - 8
		upY := y
		downY := y + 12
		ebitenutil.DebugPrintAt(screen, "^", btnX, upY)
		ebitenutil.DebugPrintAt(screen, "v", btnX, downY)
	}
}
