package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	leftPanelWidth = 200
)

type EntityPanel struct {
	panelBgImg *ebiten.Image
}

func NewEntityPanel() *EntityPanel {
	bg := ebiten.NewImage(1, 1)
	bg.Fill(color.RGBA{0x0b, 0x14, 0x2a, 0xff}) // dark blue

	return &EntityPanel{
		panelBgImg: bg,
	}
}

func (ep *EntityPanel) Update() {

}

func (ep *EntityPanel) Draw(screen *ebiten.Image) {
	// Left-side entities panel
	lpOp := &ebiten.DrawImageOptions{}
	lpOp.GeoM.Scale(float64(leftPanelWidth), float64(screen.Bounds().Dy()))
	lpOp.GeoM.Translate(0, 0)
	screen.DrawImage(ep.panelBgImg, lpOp)
	ebitenutil.DebugPrintAt(screen, "Entities:", 8, 8)
}
