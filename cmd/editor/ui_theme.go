package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// solidNineSlice returns a solid color *image.NineSlice for widget backgrounds.
func solidNineSlice(c color.Color) *image.NineSlice {
	return image.NewNineSliceColor(c)
}

func newEditorTheme(fontFace *text.Face) *widget.Theme {
	return &widget.Theme{
		ListTheme: &widget.ListParams{
			EntryFace: fontFace,
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
			BackgroundImage: solidNineSlice(color.RGBA{40, 40, 40, 255}),
		},
		ButtonTheme: &widget.ButtonParams{
			Image: &widget.ButtonImage{
				Idle:    solidNineSlice(color.RGBA{180, 180, 180, 255}),
				Hover:   solidNineSlice(color.RGBA{200, 200, 200, 255}),
				Pressed: solidNineSlice(color.RGBA{160, 160, 160, 255}),
			},
			TextFace: fontFace,
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
}
