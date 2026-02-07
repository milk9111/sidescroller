package main

import (
	"image/png"
	"os"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// TilesetPanel holds the state for the right panel tileset UI.
type TilesetPanel struct {
	Assets   []AssetInfo
	Selected int
	TileImg  *ebiten.Image
}

// NewTilesetPanel creates a widget.List for asset selection.
func NewTilesetPanel(assets []AssetInfo) *widget.List {
	entries := make([]any, len(assets))
	for i, a := range assets {
		entries[i] = a
	}
	list := widget.NewList(
		widget.ListOpts.Entries(entries),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if asset, ok := e.(AssetInfo); ok {
				return asset.Name
			}
			return ""
		}),
	)
	return list
}

// LoadTileset loads the selected asset as an ebiten.Image.
func (p *TilesetPanel) LoadTileset() error {
	if p.Selected < 0 || p.Selected >= len(p.Assets) {
		return nil
	}
	f, err := os.Open(p.Assets[p.Selected].Path)
	if err != nil {
		return err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return err
	}
	p.TileImg = ebiten.NewImageFromImage(img)
	return nil
}
