package main

import (
	"fmt"
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	leftPanelWidth = 200
)

type EntityPanel struct {
	panelBgImg *ebiten.Image
	// double-click state
	lastClickTime time.Time
	lastClickIdx  int
}

func NewEntityPanel() *EntityPanel {
	bg := ebiten.NewImage(1, 1)
	bg.Fill(color.RGBA{0x0b, 0x14, 0x2a, 0xff}) // dark blue

	return &EntityPanel{
		panelBgImg:   bg,
		lastClickIdx: -1,
	}
}

// Update handles input for the layers panel. Called from Editor.Update.
func (ep *EntityPanel) Update(g *Editor) {
	if g == nil || g.level == nil {
		return
	}

	mx, my := ebiten.CursorPosition()
	listX := 8
	listY := 28
	itemH := 28
	// only handle clicks that occur inside the left panel region
	if mx < 0 || mx >= leftPanelWidth {
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// iterate visually from top to bottom so clicks match display
		for idx := len(g.level.Layers) - 1; idx >= 0; idx-- {
			y := listY + (len(g.level.Layers)-1-idx)*itemH
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
			if mx >= listX && mx <= leftPanelWidth-40 && my >= y && my <= y+itemH {
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

	ebitenutil.DebugPrintAt(screen, "Layers:", 8, 8)

	if g == nil || g.level == nil {
		return
	}

	listX := 8
	listY := 28
	itemH := 28

	// draw layers top-first, showing visual position numbers (1., 2., ...)
	n := len(g.level.Layers)
	displayOrder := make([]int, 0, n)
	for i := n - 1; i >= 0; i-- {
		displayOrder = append(displayOrder, i)
	}
	for pos, idx := range displayOrder {
		y := listY + pos*itemH
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
