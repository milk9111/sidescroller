package main

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets/*
var assetsFS embed.FS

// PlayerTemplateSheet is the embedded player sprite sheet as an *ebiten.Image.
var PlayerTemplateSheet *ebiten.Image
var PlayerSheet *ebiten.Image

func init() {
	PlayerTemplateSheet = loadImageFromAssets("assets/player_template-Sheet.png")
	PlayerSheet = loadImageFromAssets("assets/player-Sheet.png")
}

func loadImageFromAssets(path string) *ebiten.Image {
	b, err := assetsFS.ReadFile(path)
	if err != nil {
		log.Fatalf("embed: read %s: %v", path, err)
	}

	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		log.Fatalf("embed: decode %s: %v", path, err)
	}
	return ebiten.NewImageFromImage(img)
}
