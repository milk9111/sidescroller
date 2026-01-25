package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/common"
)

func main() {
	fullscreen := flag.Bool("fullscreen", false, "run in fullscreen")
	levelPath := flag.String("level", "", "path to level JSON file")
	flag.Parse()

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("sidescroller")
	if !*fullscreen {
		ebiten.SetWindowSize(common.BaseWidth, common.BaseHeight)
	} else {
		ebiten.SetFullscreen(true)
	}

	game := NewGame(*levelPath)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
