package main

import (
	"flag"
	"log"
	"os"
	"time"

	g "github.com/AllenDang/giu"

	"github.com/milk9111/sidescroller/cmd/solum/app"
	coreautotile "github.com/milk9111/sidescroller/internal/editorcore/autotile"
	coreio "github.com/milk9111/sidescroller/internal/editorcore/io"
	coremodel "github.com/milk9111/sidescroller/internal/editorcore/model"
	"github.com/milk9111/sidescroller/internal/profiler"
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

	profilerInstance, err := profiler.Start(profiler.Config{
		Label:              "solum",
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
		if stopErr := profilerInstance.Stop(); stopErr != nil {
			log.Printf("stop profiler: %v", stopErr)
		}
	}()

	workspaceRoot, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	assets, err := coreio.ScanPNGAssets(workspaceRoot, assetDir)
	if err != nil {
		log.Fatalf("scan assets: %v", err)
	}
	prefabs, err := coreio.ScanPrefabCatalog(workspaceRoot)
	if err != nil {
		log.Fatalf("scan prefabs: %v", err)
	}

	var autotileRemap []int
	if autotileMap != "" {
		autotileRemap, err = coreautotile.LoadRemap(autotileMap)
		if err != nil {
			log.Fatalf("load autotile remap: %v", err)
		}
	}

	var doc *coremodel.LevelDocument
	saveTarget := coreio.NormalizeLevelTarget(levelName)
	loadedLevel := saveTarget
	if levelName != "" {
		doc, saveTarget, err = coreio.LoadLevel(workspaceRoot, levelName)
		if err != nil {
			log.Fatalf("load level: %v", err)
		}
		loadedLevel = saveTarget
	} else {
		doc = coremodel.NewLevelDocument(40, 22)
		saveTarget = "untitled.json"
		loadedLevel = ""
	}

	application, err := app.New(app.Config{
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

	state := application.State()
	window := g.NewMasterWindow(state.WindowTitle, 1440, 900, 0)
	window.Run(application.Loop)
}
