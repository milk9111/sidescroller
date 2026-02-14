package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	allAbilities := flag.Bool("ab", false, "start with all abilities unlocked")
	debug := flag.Bool("debug", false, "enable debug mode")
	prefabWatch := flag.Bool("watcher", false, "enable prefab hot-reload watcher")
	baseMonitor := flag.Bool("m", false, "use base monitor instead of primary (for multi-monitor setups)")
	levelName := flag.String("level", "disposal_1.json", "level name in levels/ (basename, .json optional)")
	flag.Parse()

	if *baseMonitor {
		ebiten.SetMonitor(ebiten.AppendMonitors(nil)[0])
	}

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	w, h := ebiten.Monitor().Size()
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowTitle("Defective")

	game := NewGame(*levelName, *debug, *allAbilities, *prefabWatch)

	// Hide the native OS cursor at game start; we draw a custom aim target when aiming.
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
