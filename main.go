package main

import (
	"errors"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs/component"
	sharedprofiler "github.com/milk9111/sidescroller/internal/profiler"
	"github.com/milk9111/sidescroller/scenes"
)

func main() {
	allAbilities := flag.Bool("ab", false, "start with all abilities unlocked")
	abilitiesFlag := flag.String("a", "", "comma-separated list of abilities to enable (options: anchor,double_jump,wall_grab)")
	debug := flag.Bool("debug", false, "enable debug mode")
	mute := flag.Bool("mute", false, "start with all game audio muted")
	prefabWatch := flag.Bool("watcher", false, "enable prefab hot-reload watcher")
	overlay := flag.Bool("o", false, "enable debug text overlay")
	baseMonitor := flag.Bool("m", false, "use base monitor instead of primary (for multi-monitor setups)")
	levelName := flag.String("level", "long_fall.json", "level name in levels/ (basename, .json optional)")
	pprofAddr := flag.String("pprof", "", "optional pprof listen address, for example localhost:6060")
	cpuProfilePath := flag.String("cpuprofile", "", "optional path to write a CPU profile")
	tracePath := flag.String("trace", "", "optional path to write a Go runtime execution trace")
	memProfilePath := flag.String("memprofile", "", "optional path to write a heap profile on exit")
	memProfileRate := flag.Int("memprofilerate", 0, "optional runtime.MemProfileRate override; 0 keeps the Go default")
	memProfileSample := flag.String("memprofile-sample", "", "optional interval for periodic heap snapshots, for example 30s")
	sceneName := flag.String("scene", "", "scene name to load")
	flag.Parse()
	levelProvided := flagWasProvided("level")

	var memProfileInterval time.Duration
	if *memProfileSample != "" {
		parsedInterval, err := time.ParseDuration(*memProfileSample)
		if err != nil {
			log.Fatalf("parse memprofile-sample: %v", err)
		}
		if parsedInterval <= 0 {
			log.Fatal("parse memprofile-sample: duration must be greater than zero")
		}
		memProfileInterval = parsedInterval
	}

	profilerInstance, err := sharedprofiler.Start(sharedprofiler.Config{
		Label:              "game",
		PprofAddr:          *pprofAddr,
		CPUProfilePath:     *cpuProfilePath,
		TracePath:          *tracePath,
		MemProfilePath:     *memProfilePath,
		MemProfileRate:     *memProfileRate,
		MemProfileInterval: memProfileInterval,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if stopErr := profilerInstance.Stop(); stopErr != nil {
			log.Printf("stop profiler: %v", stopErr)
		}
	}()

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

	gameConfig := scenes.GameConfig{
		LevelName:        *levelName,
		Debug:            *debug,
		AllAbilities:     *allAbilities,
		WatchPrefabs:     *prefabWatch,
		Overlay:          *overlay,
		Mute:             *mute,
		InitialAbilities: initialAbilities,
	}

	initialScene := scenes.SceneIntro
	requestedScene := strings.TrimSpace(*sceneName)
	if requestedScene != "" {
		initialScene = requestedScene
	}

	if levelProvided {
		initialScene = scenes.SceneGame
	}
	gameConfig.InitialFadeIn = initialScene == scenes.SceneIntro

	game, err := scenes.NewManager(initialScene, map[string]scenes.Factory{
		scenes.SceneGame: func() (scenes.Scene, error) {
			return scenes.NewGameScene(gameConfig), nil
		},
		scenes.SceneTest: func() (scenes.Scene, error) {
			return scenes.NewTestScene(), nil
		},
		scenes.SceneIntro: func() (scenes.Scene, error) {
			return scenes.NewIntroScene(), nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Hide the native OS cursor at game start; we draw a custom aim target when aiming.
	ebiten.SetCursorMode(ebiten.CursorModeHidden)

	if err := ebiten.RunGame(game); err != nil {
		if errors.Is(err, scenes.ErrQuit) {
			return
		}
		log.Printf("game error: %v", err)
		return
	}
}

func flagWasProvided(name string) bool {
	provided := false
	flag.CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
}
