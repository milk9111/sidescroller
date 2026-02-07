package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	frames int
}

func NewGame(levelPath string, debug bool, allAbilities bool) *Game {
	return &Game{}
}

func (g *Game) Update() error {
	g.frames++

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {

}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return outsideWidth, outsideHeight
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("shouldn't use Layout")
}
