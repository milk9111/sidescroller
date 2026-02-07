package main

import (
	"bytes"
	"image/color"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

// BuildEditorUI creates the root UI container with a right panel for the asset list.
// solidNineSlice returns a solid color *image.NineSlice for widget backgrounds.
func solidNineSlice(c color.Color) *image.NineSlice {
	return image.NewNineSliceColor(c)
}

func BuildEditorUI(assets []AssetInfo, onAssetSelected func(asset AssetInfo, setTileset func(img *ebiten.Image))) *ebitenui.UI {
	ui := &ebitenui.UI{}

	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic("Failed to load font: " + err.Error())
	}

	var fontFace text.Face = &text.GoTextFace{Source: s, Size: 14}

	ui.PrimaryTheme = &widget.Theme{
		ListTheme: &widget.ListParams{
			EntryFace: &fontFace,
			EntryColor: &widget.ListEntryColor{
				Unselected:          color.Black,
				Selected:            color.RGBA{0, 0, 128, 255},
				DisabledUnselected:  color.Gray{Y: 128},
				DisabledSelected:    color.Gray{Y: 64},
				SelectingBackground: color.RGBA{200, 220, 255, 255},
				SelectedBackground:  color.RGBA{180, 200, 255, 255},
			},
			ScrollContainerImage: &widget.ScrollContainerImage{
				Idle: solidNineSlice(color.RGBA{220, 220, 220, 255}),
				Mask: solidNineSlice(color.RGBA{220, 220, 220, 255}),
			},
		},
		PanelTheme: &widget.PanelParams{
			BackgroundImage: solidNineSlice(color.RGBA{240, 240, 240, 255}),
		},
		ButtonTheme: &widget.ButtonParams{
			Image: &widget.ButtonImage{
				Idle:    solidNineSlice(color.RGBA{180, 180, 180, 255}),
				Hover:   solidNineSlice(color.RGBA{200, 200, 200, 255}),
				Pressed: solidNineSlice(color.RGBA{160, 160, 160, 255}),
			},
			TextFace: &fontFace,
			TextColor: &widget.ButtonTextColor{
				Idle: color.Black,
			},
		},
		SliderTheme: &widget.SliderParams{
			TrackImage: &widget.SliderTrackImage{
				Idle:  solidNineSlice(color.RGBA{180, 180, 180, 255}),
				Hover: solidNineSlice(color.RGBA{200, 200, 200, 255}),
			},
			HandleImage: &widget.ButtonImage{
				Idle:    solidNineSlice(color.RGBA{120, 120, 120, 255}),
				Hover:   solidNineSlice(color.RGBA{160, 160, 160, 255}),
				Pressed: solidNineSlice(color.RGBA{100, 100, 100, 255}),
			},
		},
	}

	var tilesetImg *ebiten.Image
	var tileGrid *widget.Container

	// Right panel: asset list
	assetPanel := widget.NewPanel(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(240, 400),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	var entries []any
	if len(assets) > 0 {
		entries = make([]any, len(assets))
		for i, a := range assets {
			entries[i] = a
		}
	} else {
		entries = []any{}
	}
	assetList := widget.NewList(
		widget.ListOpts.Entries(entries),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if asset, ok := e.(AssetInfo); ok {
				return asset.Name
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			if asset, ok := args.Entry.(AssetInfo); ok {
				onAssetSelected(asset, func(img *ebiten.Image) {
					tilesetImg = img
					// Replace or add the tile grid below the asset list
					if tileGrid != nil {
						assetPanel.RemoveChild(tileGrid)
					}
					tileGrid = NewTilesetGrid(tilesetImg, 32, func(tileIndex int) {
						// TODO: handle tile selection
					})
					assetPanel.AddChild(tileGrid)
				})
			}
		}),
	)
	assetPanel.AddChild(assetList)

	// Root container: just the right panel for now
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	root.AddChild(assetPanel)

	ui.Container = root
	return ui
}
