package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	allAbilities := flag.Bool("ab", false, "start with all abilities unlocked")
	debug := flag.Bool("debug", false, "enable debug mode")
	levelName := flag.String("level", "", "level name in levels/ (basename, .json optional)")
	flag.Parse()

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("sidescroller")

	game := NewGame(*levelName, *debug, *allAbilities)

	// Hide the native OS cursor at game start; we draw a custom aim target when aiming.
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	ebiten.SetFullscreen(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
