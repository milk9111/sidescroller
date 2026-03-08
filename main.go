package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs/component"
)

var ErrQuit = errors.New("quit")

func main() {
	allAbilities := flag.Bool("ab", false, "start with all abilities unlocked")
	abilitiesFlag := flag.String("a", "", "comma-separated list of abilities to enable (options: anchor,double_jump,wall_grab)")
	debug := flag.Bool("debug", false, "enable debug mode")
	prefabWatch := flag.Bool("watcher", false, "enable prefab hot-reload watcher")
	overlay := flag.Bool("o", false, "enable debug text overlay")
	baseMonitor := flag.Bool("m", false, "use base monitor instead of primary (for multi-monitor setups)")
	levelName := flag.String("level", "long_fall.json", "level name in levels/ (basename, .json optional)")
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

	// Build initial abilities from -a (unless -ab is set, which enables all)
	var initialAbilities *component.Abilities
	if *allAbilities {
		initialAbilities = &component.Abilities{Anchor: true, DoubleJump: true, WallGrab: true}
	} else if *abilitiesFlag != "" {
		a := &component.Abilities{}
		seen := map[string]bool{}
		for _, raw := range strings.Split(*abilitiesFlag, ",") {
			s := strings.TrimSpace(strings.ToLower(raw))
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			switch s {
			case "anchor":
				a.Anchor = true
			case "double_jump":
				a.DoubleJump = true
			case "wall_grab":
				a.WallGrab = true
			}
		}
		initialAbilities = a
	}

	game := NewGame(*levelName, *debug, *allAbilities, *prefabWatch, *overlay, initialAbilities)

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
