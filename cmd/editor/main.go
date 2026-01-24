package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	file := flag.String("file", "", "level file to open (optional)")
	flag.Parse()

	// Editor uses fixed grid: 40x23 cells at 32px each.
	eg := NewEditor(32)
	if *file != "" {
		if err := eg.Load(*file); err != nil {
			log.Printf("failed to load %s: %v", *file, err)
			// fall back to init
			eg.Init(40, 23)
		}
	} else {
		eg.Init(40, 23)
	}

	// Window matches grid pixel size: 40*32 x 23*32 = 1280x736
	ebiten.SetWindowSize(baseWidthEditor, baseHeightEditor)
	ebiten.SetWindowTitle("sidescroller - editor")
	if err := ebiten.RunGame(eg); err != nil {
		log.Fatal(err)
	}
}
