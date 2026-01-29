package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/common"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug mode")
	fullscreen := flag.Bool("fullscreen", false, "run in fullscreen")
	levelPath := flag.String("level", "", "path to level JSON file")
	flag.Parse()

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("sidescroller")

	game := NewGame(*levelPath, *debug)

	if !*fullscreen {
		if game != nil && game.level != nil {
			w := game.level.Width * common.TileSize
			h := game.level.Height * common.TileSize
			ebiten.SetWindowSize(w, h)
		} else {
			ebiten.SetWindowSize(common.BaseWidth, common.BaseHeight)
		}
	} else {
		ebiten.SetFullscreen(true)
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
