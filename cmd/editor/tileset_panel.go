package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	rightPanelWidth = 220
)

// TilesetPanel encapsulates tileset panel state and behavior.
type TilesetPanel struct {
	X          int
	Y          int
	W          int
	H          int
	Zoom       float64
	OffsetX    float64
	OffsetY    float64
	DragActive bool
	LastMX     int
	LastMY     int
	Hover      int

	assetList    []string
	tilesetPath  string
	tilesetImg   *ebiten.Image
	tilesetTileW int
	tilesetTileH int
	tilesetCols  int
	selectedTile int // 0-based index
	cellSize     int

	panelBgImg      *ebiten.Image
	hoverBorderImg  *ebiten.Image
	selectBorderImg *ebiten.Image
}

// NewTilesetPanel constructs a TilesetPanel with provided geometry and zoom.
func NewTilesetPanel(w, h, cellSize int, zoom float64) *TilesetPanel {
	var assetList []string

	// populate embedded asset list from assets/ (if available)
	if entries, err := os.ReadDir("assets"); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				name := e.Name()
				if len(name) > 4 && (name[len(name)-4:] == ".png" || name[len(name)-4:] == ".PNG") {
					assetList = append(assetList, name)
				}
			}
		}
	}

	panelX := baseWidthEditor - rightPanelWidth
	// panel background (1x1) and hover/select borders
	bg := ebiten.NewImage(1, 1)
	bg.Fill(color.RGBA{0x0b, 0x14, 0x2a, 0xff}) // dark blue
	hb := ebiten.NewImage(1, 1)
	hb.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
	sb := ebiten.NewImage(1, 1)
	sb.Fill(color.RGBA{0xff, 0xd7, 0x00, 0xff})

	return &TilesetPanel{
		X:               panelX + 8,
		Y:               8 + len(assetList)*18 + 8,
		W:               w,
		H:               h,
		Zoom:            zoom,
		Hover:           -1,
		assetList:       assetList,
		cellSize:        cellSize,
		selectedTile:    -1,
		panelBgImg:      bg,
		hoverBorderImg:  hb,
		selectBorderImg: sb,
	}
}

// Update handles tileset panel interactions; returns true if cursor is over the panel.
func (tp *TilesetPanel) Update(mx, my, panelX int, leftPressed bool, prevMouse bool) bool {
	tp.X = panelX + 8
	inPanel := mx >= tp.X && mx < tp.X+tp.W && my >= tp.Y && my < tp.Y+tp.H

	// wheel zoom (centered on mouse)
	if inPanel {
		_, wy := ebiten.Wheel()
		if wy != 0 {
			// compute local tile-space coordinate before zoom
			localX := (float64(mx) - float64(tp.X) - 8 - tp.OffsetX) / tp.Zoom
			localY := (float64(my) - float64(tp.Y) - 8 - tp.OffsetY) / tp.Zoom
			var factor float64
			if wy > 0 {
				factor = 1.1
			} else {
				factor = 1.0 / 1.1
			}
			newZoom := tp.Zoom * factor
			if newZoom < 0.25 {
				newZoom = 0.25
			}
			if newZoom > 4.0 {
				newZoom = 4.0
			}
			tp.Zoom = newZoom
			// recompute offset so point under cursor stays fixed
			tp.OffsetX = float64(mx) - float64(tp.X) - 8 - localX*tp.Zoom
			tp.OffsetY = float64(my) - float64(tp.Y) - 8 - localY*tp.Zoom
		}
	}

	// right-button drag to pan
	rPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	if rPressed {
		if !tp.DragActive && inPanel {
			tp.DragActive = true
			tp.LastMX = mx
			tp.LastMY = my
		}
		if tp.DragActive {
			dx := mx - tp.LastMX
			dy := my - tp.LastMY
			tp.OffsetX += float64(dx)
			tp.OffsetY += float64(dy)
			tp.LastMX = mx
			tp.LastMY = my
		}
	} else {
		tp.DragActive = false
	}

	// clamp offsets so tileset content cannot be dragged completely out of the panel
	if tp.tilesetTileW > 0 && tp.tilesetTileH > 0 && tp.tilesetImg != nil {
		cols := tp.tilesetImg.Bounds().Dx() / tp.tilesetTileW
		rows := tp.tilesetImg.Bounds().Dy() / tp.tilesetTileH
		contentW := float64(cols) * float64(tp.tilesetTileW) * tp.Zoom
		contentH := float64(rows) * float64(tp.tilesetTileH) * tp.Zoom
		innerW := float64(tp.W - 16)
		innerH := float64(tp.H - 16)
		// min offset so right/bottom edges still cover panel
		minX := math.Min(0, innerW-contentW)
		minY := math.Min(0, innerH-contentH)
		if tp.OffsetX < minX {
			tp.OffsetX = minX
		}
		if tp.OffsetY < minY {
			tp.OffsetY = minY
		}
		if tp.OffsetX > 0 {
			tp.OffsetX = 0
		}
		if tp.OffsetY > 0 {
			tp.OffsetY = 0
		}
	}

	// compute hover tile under mouse (even without clicks)
	tp.Hover = -1
	if inPanel && tp.tilesetTileW > 0 && tp.tilesetTileH > 0 && tp.tilesetImg != nil {
		localX := (float64(mx) - float64(tp.X) - 8 - tp.OffsetX) / (float64(tp.tilesetTileW) * tp.Zoom)
		localY := (float64(my) - float64(tp.Y) - 8 - tp.OffsetY) / (float64(tp.tilesetTileH) * tp.Zoom)
		if localX >= 0 && localY >= 0 {
			col := int(math.Floor(localX))
			row := int(math.Floor(localY))
			cols := tp.tilesetImg.Bounds().Dx() / tp.tilesetTileW
			rows := tp.tilesetImg.Bounds().Dy() / tp.tilesetTileH
			if col >= 0 && row >= 0 && col < cols && row < rows {
				tp.Hover = row*cols + col
			}
		}
	}

	// left-click selection in tileset panel
	if leftPressed && !prevMouse && tp.Hover >= 0 {
		tp.selectedTile = tp.Hover
	}

	// Right-side panel for tileset and assets (split into file list + a draggable, zoomable tileset panel)
	// asset list area (click to load an asset)
	listStartY := 8
	lineH := 18
	if leftPressed && !prevMouse {
		for i, name := range tp.assetList {
			y0 := listStartY + i*lineH
			listX := panelX + 8
			listW := rightPanelWidth - 16
			if mx >= listX && mx < listX+listW && my >= y0 && my < y0+16 {
				if b, err := os.ReadFile(filepath.Join("assets", name)); err == nil {
					if img, _, err := image.Decode(bytes.NewReader(b)); err == nil {
						tp.tilesetImg = ebiten.NewImageFromImage(img)
						// default tiles size to cellSize unless already specified
						if tp.tilesetTileW == 0 {
							tp.tilesetTileW = tp.cellSize
						}
						if tp.tilesetTileH == 0 {
							tp.tilesetTileH = tp.cellSize
						}
						if tp.tilesetTileW > 0 {
							tp.tilesetCols = tp.tilesetImg.Bounds().Dx() / tp.tilesetTileW
						}
						tp.tilesetPath = name
						tp.selectedTile = 0
					}
				}
				break
			}
		}
	}

	return inPanel
}

// Draw renders the tileset panel.
func (tp *TilesetPanel) Draw(screen *ebiten.Image, panelX int) {
	// Right-side panel background
	rpOp := &ebiten.DrawImageOptions{}
	rpOp.GeoM.Scale(float64(rightPanelWidth), float64(screen.Bounds().Dy()))
	rpOp.GeoM.Translate(float64(panelX), 0)
	screen.DrawImage(tp.panelBgImg, rpOp)

	tp.X = panelX + 8
	// draw panel background
	bgOp := &ebiten.DrawImageOptions{}
	bgOp.GeoM.Scale(float64(tp.W), float64(tp.H))
	bgOp.GeoM.Translate(float64(tp.X), float64(tp.Y))
	screen.DrawImage(tp.panelBgImg, bgOp)

	// draw tileset grid only when an image is loaded
	if tp.tilesetImg != nil {
		cols := 1
		if tp.tilesetTileW > 0 {
			cols = tp.tilesetImg.Bounds().Dx() / tp.tilesetTileW
		}
		rows := 1
		if tp.tilesetTileH > 0 {
			rows = tp.tilesetImg.Bounds().Dy() / tp.tilesetTileH
		}
		tileWf := float64(tp.tilesetTileW) * tp.Zoom
		tileHf := float64(tp.tilesetTileH) * tp.Zoom
		// draw tiles
		for ry := 0; ry < rows; ry++ {
			for rx := 0; rx < cols; rx++ {
				idx := ry*cols + rx
				sx := rx * tp.tilesetTileW
				sy := ry * tp.tilesetTileH
				r := image.Rect(sx, sy, sx+tp.tilesetTileW, sy+tp.tilesetTileH)
				sub := tp.tilesetImg.SubImage(r).(*ebiten.Image)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(tp.Zoom, tp.Zoom)
				dx := float64(tp.X+8) + tp.OffsetX + float64(rx)*tileWf
				dy := float64(tp.Y+8) + tp.OffsetY + float64(ry)*tileHf
				op.GeoM.Translate(dx, dy)
				screen.DrawImage(sub, op)

				// hover border
				if tp.Hover == idx {
					hbOp := &ebiten.DrawImageOptions{}
					hbOp.GeoM.Scale(tileWf, 1)
					hbOp.GeoM.Translate(dx, dy)
					screen.DrawImage(tp.hoverBorderImg, hbOp)
					hbOp2 := &ebiten.DrawImageOptions{}
					hbOp2.GeoM.Scale(tileWf, 1)
					hbOp2.GeoM.Translate(dx, dy+tileHf-1)
					screen.DrawImage(tp.hoverBorderImg, hbOp2)
					hbOp3 := &ebiten.DrawImageOptions{}
					hbOp3.GeoM.Scale(1, tileHf)
					hbOp3.GeoM.Translate(dx, dy)
					screen.DrawImage(tp.hoverBorderImg, hbOp3)
					hbOp4 := &ebiten.DrawImageOptions{}
					hbOp4.GeoM.Scale(1, tileHf)
					hbOp4.GeoM.Translate(dx+tileWf-1, dy)
					screen.DrawImage(tp.hoverBorderImg, hbOp4)
				}

				// selected border
				if tp.selectedTile == idx {
					sbOp := &ebiten.DrawImageOptions{}
					sbOp.GeoM.Scale(tileWf, 1)
					sbOp.GeoM.Translate(dx, dy)
					screen.DrawImage(tp.selectBorderImg, sbOp)
					sbOp2 := &ebiten.DrawImageOptions{}
					sbOp2.GeoM.Scale(tileWf, 1)
					sbOp2.GeoM.Translate(dx, dy+tileHf-1)
					screen.DrawImage(tp.selectBorderImg, sbOp2)
					sbOp3 := &ebiten.DrawImageOptions{}
					sbOp3.GeoM.Scale(1, tileHf)
					sbOp3.GeoM.Translate(dx, dy)
					screen.DrawImage(tp.selectBorderImg, sbOp3)
					sbOp4 := &ebiten.DrawImageOptions{}
					sbOp4.GeoM.Scale(1, tileHf)
					sbOp4.GeoM.Translate(dx+tileWf-1, dy)
					screen.DrawImage(tp.selectBorderImg, sbOp4)
				}
			}
		}
	}

	// always draw asset list so user can load a tileset
	y := 8
	for i, name := range tp.assetList {
		ebitenutil.DebugPrintAt(screen, name, panelX+8, y+i*18)
	}

	// show current tileset settings
	infoY := baseHeightEditor - 80
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Tileset: %s", tp.tilesetPath), panelX+8, infoY)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("TileSize: %dx%d", tp.tilesetTileW, tp.tilesetTileH), panelX+8, infoY+18)
}
