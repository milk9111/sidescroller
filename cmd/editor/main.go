package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	pprof := flag.Bool("pprof", false, "enable pprof HTTP server")
	file := flag.String("file", "", "level file to open (optional)")
	flag.Parse()

	// Editor uses fixed grid: 40x23 cells at 32px each.
	eg := NewEditor(32, *pprof)
	if *file != "" {
		if err := eg.Load(*file); err != nil {
			log.Printf("failed to load %s: %v", *file, err)
			// fall back to init
			eg.Init(40, 23)
		}
	} else {
		eg.Init(40, 23)
	}

	// Window matches the current screen size
	sw, sh := ebiten.ScreenSizeInFullscreen()
	ebiten.SetWindowSize(sw, sh)
	ebiten.SetWindowDecorated(true)
	ebiten.SetWindowTitle("sidescroller - editor")
	if err := ebiten.RunGame(eg); err != nil {
		log.Fatal(err)
	}
}
