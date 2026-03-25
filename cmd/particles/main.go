package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	sharedprofiler "github.com/milk9111/sidescroller/internal/profiler"
)

func main() {
	var assetDir string
	var prefabDir string
	var file string
	var pprofAddr string
	var cpuProfilePath string
	var tracePath string
	var memProfilePath string
	var memProfileRate int
	var memProfileSample string
	flag.StringVar(&assetDir, "asset-dir", "assets", "directory scanned recursively for particle images")
	flag.StringVar(&prefabDir, "prefab-dir", "prefabs", "directory to save particle emitter prefabs into")
	flag.StringVar(&file, "file", "", "optional prefab file name under prefabs/ to load into the particle editor")
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

	profilerInstance, err := sharedprofiler.Start(sharedprofiler.Config{
		Label:              "particles",
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

	assets, err := editorio.ScanPNGAssets(workspaceRoot, assetDir)
	if err != nil {
		log.Fatalf("scan assets: %v", err)
	}

	app, err := NewApp(AppConfig{
		WorkspaceRoot: workspaceRoot,
		AssetDir:      assetDir,
		PrefabDir:     prefabDir,
		File:          file,
		Assets:        assets,
	})
	if err != nil {
		log.Fatal(err)
	}

	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	w, h := ebiten.Monitor().Size()
	ebiten.SetWindowSize(w, h)
	ebiten.SetRunnableOnUnfocused(false)
	ebiten.SetWindowTitle("Defective Particle Emitter Tool")
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ErrQuit) {
		log.Fatal(err)
	}
}
