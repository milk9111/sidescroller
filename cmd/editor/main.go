package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/milk9111/sidescroller/cmd/editor/autotile"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
)

func main() {
	var assetDir string
	var levelName string
	var autotileMap string
	var pprofAddr string
	var cpuProfilePath string
	var tracePath string
	var memProfilePath string
	var memProfileRate int
	var memProfileSample string
	flag.StringVar(&assetDir, "dir", "assets", "directory scanned recursively for tileset images")
	flag.StringVar(&levelName, "level", "", "optional level file to load from levels/")
	flag.StringVar(&autotileMap, "autotile-map", "", "optional autotile remap JSON file")
	flag.StringVar(&pprofAddr, "pprof", "", "optional pprof listen address, for example localhost:6060")
	flag.StringVar(&cpuProfilePath, "cpuprofile", "", "optional path to write a CPU profile")
	flag.StringVar(&tracePath, "trace", "", "optional path to write a Go runtime execution trace")
	flag.StringVar(&memProfilePath, "memprofile", "", "optional path to write a heap profile on exit")
	flag.IntVar(&memProfileRate, "memprofilerate", 0, "optional runtime.MemProfileRate override; 0 keeps the Go default")
	flag.StringVar(&memProfileSample, "memprofile-sample", "", "optional interval for periodic heap snapshots, for example 30s")
	flag.Parse()

	var memProfileInterval time.Duration
	if memProfileSample != "" {
		parsedInterval, err := time.ParseDuration(memProfileSample)
		if err != nil {
			log.Fatalf("parse memprofile-sample: %v", err)
		}
		if parsedInterval <= 0 {
			log.Fatal("parse memprofile-sample: duration must be greater than zero")
		}
		memProfileInterval = parsedInterval
	}

	profiler, err := startProfiler(profilerConfig{
		PprofAddr:          pprofAddr,
		CPUProfilePath:     cpuProfilePath,
		TracePath:          tracePath,
		MemProfilePath:     memProfilePath,
		MemProfileRate:     memProfileRate,
		MemProfileInterval: memProfileInterval,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if stopErr := profiler.Stop(); stopErr != nil {
			log.Printf("stop profiler: %v", stopErr)
		}
	}()

	workspaceRoot, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	assets, err := editorio.ScanPNGAssets(workspaceRoot, assetDir)
	if err != nil {
		log.Fatalf("scan assets: %v", err)
	}
	prefabs, err := editorio.ScanPrefabCatalog(workspaceRoot)
	if err != nil {
		log.Fatalf("scan prefabs: %v", err)
	}

	var autotileRemap []int
	if autotileMap != "" {
		autotileRemap, err = autotile.LoadRemap(autotileMap)
		if err != nil {
			log.Fatalf("load autotile remap: %v", err)
		}
	}

	var doc *model.LevelDocument
	saveTarget := editorio.NormalizeLevelTarget(levelName)
	loadedLevel := saveTarget
	if levelName != "" {
		doc, saveTarget, err = editorio.LoadLevel(workspaceRoot, levelName)
		if err != nil {
			log.Fatalf("load level: %v", err)
		}
		loadedLevel = saveTarget
	} else {
		defaultWidth, defaultHeight := defaultLevelSize()
		width, height, promptErr := editorio.PromptForLevelSize(defaultWidth, defaultHeight)
		if promptErr != nil {
			log.Fatalf("prompt level size: %v", promptErr)
		}
		doc = model.NewLevelDocument(width, height)
		saveTarget = "untitled.json"
		loadedLevel = ""
	}

	app, err := NewApp(AppConfig{
		WorkspaceRoot: workspaceRoot,
		AssetDir:      assetDir,
		LevelName:     loadedLevel,
		SaveTarget:    saveTarget,
		Level:         doc,
		Assets:        assets,
		Prefabs:       prefabs,
		AutotileRemap: autotileRemap,
	})
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	w, h := ebiten.Monitor().Size()
	ebiten.SetWindowSize(w, h)
	ebiten.SetRunnableOnUnfocused(false)
	ebiten.SetWindowTitle("Defective Editor")
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ErrQuit) {
		log.Fatal(err)
	}
}

func defaultLevelSize() (int, int) {
	width, height := ebiten.ScreenSizeInFullscreen()
	if width <= 0 || height <= 0 {
		return 40, 22
	}
	return max(10, width/model.DefaultTileSize), max(8, height/model.DefaultTileSize)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
