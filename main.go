package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug mode")
	levelPath := flag.String("level", "", "path to level JSON file")
	flag.Parse()

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("sidescroller")

	game := NewGame(*levelPath, *debug)

	// Hide the native OS cursor at game start; we draw a custom aim target when aiming.
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	ebiten.SetFullscreen(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
