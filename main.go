package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/hajimehoshi/ebiten/v2"
)

var ErrQuit = errors.New("quit")

func main() {
	allAbilities := flag.Bool("ab", false, "start with all abilities unlocked")
	debug := flag.Bool("debug", false, "enable debug mode")
	prefabWatch := flag.Bool("watcher", false, "enable prefab hot-reload watcher")
	baseMonitor := flag.Bool("m", false, "use base monitor instead of primary (for multi-monitor setups)")
	levelName := flag.String("level", "disposal_1.json", "level name in levels/ (basename, .json optional)")
	profile := flag.Bool("profile", false, "start http server exposing pprof endpoints on localhost:6060")
	flag.Parse()

	if *profile {
		go func() {
			log.Println("pprof HTTP server listening on http://localhost:6060/debug/pprof/")
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				log.Printf("pprof server error: %v", err)
			}
		}()
	}

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
		if err == ErrQuit {
			return
		}
		log.Printf("game error: %v", err)
		return
	}
}
