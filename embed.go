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

func init() {
    b, err := assetsFS.ReadFile("assets/player_template-Sheet.png")
    if err != nil {
        log.Fatalf("embed: read player_template-Sheet.png: %v", err)
    }

    img, _, err := image.Decode(bytes.NewReader(b))
    if err != nil {
        log.Fatalf("embed: decode player_template-Sheet.png: %v", err)
    }

    PlayerTemplateSheet = ebiten.NewImageFromImage(img)
}
