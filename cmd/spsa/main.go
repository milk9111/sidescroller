package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/obj"
)

type demoGame struct {
	player *obj.Player
}

func (g *demoGame) Update() error {
	// Keep player idle; call Update so animation advances
	g.player.Update()
	return nil
}

func (g *demoGame) Draw(screen *ebiten.Image) {
	// position player in center of window
	w, h := 512, 512
	// center using RenderWidth/Height
	gx := float32((w - int(g.player.RenderWidth)) / 2)
	gy := float32((h - int(g.player.RenderHeight)) / 2)
	g.player.X = gx
	g.player.Y = gy
	g.player.Draw(screen)
}

func (g *demoGame) Layout(outsideWidth, outsideHeight int) (int, int) { return 512, 512 }

func main() {
	lvl := &obj.Level{Width: 40, Height: 23}
	cw := obj.NewCollisionWorld(lvl)
	input := obj.NewInput()
	player := obj.NewPlayer(0, 0, input, cw)
	ebiten.SetWindowSize(512, 512)
	ebiten.SetWindowTitle("Player Idle Demo")
	game := &demoGame{player: player}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
