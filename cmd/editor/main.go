package main

import (
	"log"
)

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/ebitenui/ebitenui"
)

// EditorGame is the Ebiten game for the editor.
type EditorGame struct {
	ui *ebitenui.UI
}

func (g *EditorGame) Update() error {
	if g.ui != nil {
		g.ui.Update()
	}
	return nil
}

func (g *EditorGame) Draw(screen *ebiten.Image) {
	if g.ui != nil {
		g.ui.Draw(screen)
	}
}

func (g *EditorGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 1280, 720 // Default editor window size
}

func main() {
	log.Println("Editor starting...")
	ui := &ebitenui.UI{}
	game := &EditorGame{ui: ui}
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
