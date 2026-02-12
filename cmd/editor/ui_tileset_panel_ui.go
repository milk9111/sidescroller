package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func buildTilesetPanelUI(
	assets []AssetInfo,
	theme *widget.Theme,
	fontFace *text.Face,
	onAssetSelected func(asset AssetInfo, setTileset func(img *ebiten.Image)),
	onTileSelected func(tileIndex int),
) *TilesetPanelUI {
	var tilesetImg *ebiten.Image
	var tileGridZoom *TilesetGridZoomable

	// Asset list entries
	entries := make([]any, 0, len(assets))
	for _, a := range assets {
		entries = append(entries, a)
	}

	// Tileset panel: vertical layout (top: asset list, bottom: tileset grid)
	tilesetPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(240, 400),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{40, 40, 40, 255})),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)

	// helper to apply a tileset image into the tileset panel
	applyTileset := func(img *ebiten.Image) {
		tilesetImg = img
		if tileGridZoom != nil {
			tilesetPanel.RemoveChild(tileGridZoom.Container)
		}
		tileGridZoom = NewTilesetGridZoomable(tilesetImg, 32, func(tileIndex int) {
			if onTileSelected != nil {
				onTileSelected(tileIndex)
			}
		})
		tilesetPanel.AddChild(tileGridZoom.Container)
	}

	setTilesetSelection := func(tileIndex int) {
		if tileGridZoom == nil {
			return
		}
		tileGridZoom.SetSelected(tileIndex)
	}

	setTilesetSelectionEnabled := func(enabled bool) {
		if tileGridZoom == nil {
			return
		}
		tileGridZoom.SetSelectionEnabled(enabled)
	}

	// Asset list (scrollable)
	assetList := widget.NewList(
		widget.ListOpts.Entries(entries),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if asset, ok := e.(AssetInfo); ok {
				return asset.Name
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			if onAssetSelected == nil {
				return
			}
			if asset, ok := args.Entry.(AssetInfo); ok {
				onAssetSelected(asset, applyTileset)
			}
		}),
	)
	tilesetPanel.AddChild(assetList)

	_ = theme
	_ = fontFace

	return &TilesetPanelUI{
		Container:                  tilesetPanel,
		ApplyTileset:               applyTileset,
		SetTilesetSelection:        setTilesetSelection,
		SetTilesetSelectionEnabled: setTilesetSelectionEnabled,
	}
}
