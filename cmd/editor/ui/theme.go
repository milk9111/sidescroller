package editorui

import (
	"bytes"
	"image/color"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

type Theme struct {
	Face              textv2.Face
	TitleFace         textv2.Face
	PanelBackground   *euiimage.NineSlice
	ToolbarBackground *euiimage.NineSlice
	ButtonImage       *widget.ButtonImage
	ActiveButtonImage *widget.ButtonImage
	InputImage        *widget.TextInputImage
	ButtonText        *widget.ButtonTextColor
	InputColor        *widget.TextInputColor
	PanelPadding      *widget.Insets
	ButtonPadding     *widget.Insets
	TextColor         color.Color
	MutedTextColor    color.Color
}

func NewTheme() (*Theme, error) {
	source, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, err
	}

	baseFace := &textv2.GoTextFace{Source: source, Size: 16}
	titleFace := &textv2.GoTextFace{Source: source, Size: 18}

	return &Theme{
		Face:              baseFace,
		TitleFace:         titleFace,
		PanelBackground:   euiimage.NewNineSliceColor(color.NRGBA{R: 24, G: 25, B: 31, A: 245}),
		ToolbarBackground: euiimage.NewNineSliceColor(color.NRGBA{R: 27, G: 29, B: 37, A: 245}),
		ButtonImage: &widget.ButtonImage{
			Idle:    euiimage.NewNineSliceColor(color.NRGBA{R: 51, G: 56, B: 70, A: 255}),
			Hover:   euiimage.NewNineSliceColor(color.NRGBA{R: 66, G: 73, B: 91, A: 255}),
			Pressed: euiimage.NewNineSliceColor(color.NRGBA{R: 41, G: 45, B: 57, A: 255}),
		},
		ActiveButtonImage: &widget.ButtonImage{
			Idle:    euiimage.NewNineSliceColor(color.NRGBA{R: 68, G: 114, B: 255, A: 255}),
			Hover:   euiimage.NewNineSliceColor(color.NRGBA{R: 92, G: 133, B: 255, A: 255}),
			Pressed: euiimage.NewNineSliceColor(color.NRGBA{R: 54, G: 97, B: 234, A: 255}),
		},
		InputImage: &widget.TextInputImage{
			Idle:     euiimage.NewNineSliceColor(color.NRGBA{R: 42, G: 45, B: 56, A: 255}),
			Disabled: euiimage.NewNineSliceColor(color.NRGBA{R: 33, G: 35, B: 44, A: 255}),
		},
		ButtonText: &widget.ButtonTextColor{
			Idle:     color.NRGBA{R: 233, G: 239, B: 255, A: 255},
			Disabled: color.NRGBA{R: 144, G: 150, B: 168, A: 255},
			Hover:    color.NRGBA{R: 255, G: 255, B: 255, A: 255},
			Pressed:  color.NRGBA{R: 250, G: 250, B: 255, A: 255},
		},
		InputColor: &widget.TextInputColor{
			Idle:          color.NRGBA{R: 240, G: 242, B: 248, A: 255},
			Disabled:      color.NRGBA{R: 160, G: 164, B: 176, A: 255},
			Caret:         color.NRGBA{R: 240, G: 242, B: 248, A: 255},
			DisabledCaret: color.NRGBA{R: 160, G: 164, B: 176, A: 255},
		},
		PanelPadding:   widget.NewInsetsSimple(12),
		ButtonPadding:  &widget.Insets{Left: 14, Right: 14, Top: 8, Bottom: 8},
		TextColor:      color.NRGBA{R: 233, G: 239, B: 255, A: 255},
		MutedTextColor: color.NRGBA{R: 176, G: 184, B: 201, A: 255},
	}, nil
}
