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
	if !*fullscreen {
		ebiten.SetWindowSize(common.BaseWidth, common.BaseHeight)
	} else {
		ebiten.SetFullscreen(true)
	}

	game := NewGame(*levelPath, *debug)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
