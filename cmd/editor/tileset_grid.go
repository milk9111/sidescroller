package main

import (
	"image"
	"image/color"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

type TilesetGridZoomable struct {
	Tileset   *ebiten.Image
	TileSize  int
	Selected  int
	Zoom      float64
	PanX      int
	PanY      int
	Container *widget.Container
}

func NewTilesetGridZoomable(tileset *ebiten.Image, tileSize int, onSelect func(tileIndex int)) *TilesetGridZoomable {
	g := &TilesetGridZoomable{
		Tileset:  tileset,
		TileSize: tileSize,
		Zoom:     1.0,
		PanX:     0,
		PanY:     0,
	}
	if tileset == nil {
		g.Container = widget.NewContainer()
		return g
	}

	w, h := tileset.Size()
	tilesX := w / tileSize
	tilesY := h / tileSize

	g.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(tilesX),
				widget.GridLayoutOpts.Spacing(2, 2),
			),
		),
	)

	var buttons []*widget.Button
	var group *widget.RadioGroup
	for y := 0; y < tilesY; y++ {
		for x := 0; x < tilesX; x++ {
			sub := tileset.SubImage(
				image.Rect(x*tileSize, y*tileSize, (x+1)*tileSize, (y+1)*tileSize),
			).(*ebiten.Image)
			idleImg := euiimage.NewAdvancedNineSliceImage(sub, euiimage.NewBorder(0, 0, 0, 0, color.Black))
			hoverImg := euiimage.NewAdvancedNineSliceImage(sub, euiimage.NewBorder(1, 1, 1, 1, color.RGBA{255, 255, 0, 255}))
			pressedImg := euiimage.NewAdvancedNineSliceImage(sub, euiimage.NewBorder(1, 1, 1, 1, color.RGBA{0, 200, 255, 255}))
			pressedHoverImg := euiimage.NewAdvancedNineSliceImage(sub, euiimage.NewBorder(1, 1, 1, 1, color.RGBA{0, 200, 255, 255}))
			idx := len(buttons)
			btn := widget.NewButton(
				widget.ButtonOpts.Image(&widget.ButtonImage{
					Idle:         idleImg,
					Hover:        hoverImg,
					Pressed:      pressedImg,
					PressedHover: pressedHoverImg,
					Disabled:     idleImg,
				}),
				widget.ButtonOpts.ToggleMode(),
				widget.ButtonOpts.WidgetOpts(
					widget.WidgetOpts.MinSize(tileSize, tileSize),
				),
				widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
					if group != nil && idx >= 0 && idx < len(buttons) {
						group.SetActive(buttons[idx])
					}
				}),
			)
			buttons = append(buttons, btn)
			g.Container.AddChild(btn)
		}
	}

	elements := make([]widget.RadioGroupElement, 0, len(buttons))
	for _, b := range buttons {
		elements = append(elements, b)
	}

	group = widget.NewRadioGroup(
		widget.RadioGroupOpts.Elements(elements...),
		widget.RadioGroupOpts.ChangedHandler(func(args *widget.RadioGroupChangedEventArgs) {
			for i, b := range buttons {
				if args.Active == b {
					g.Selected = i
					if onSelect != nil {
						onSelect(i)
					}
					return
				}
			}
		}),
	)
	if len(buttons) > 0 {
		group.SetActive(buttons[0])
		g.Selected = 0
		if onSelect != nil {
			onSelect(0)
		}
	}
	return g
}

func (g *TilesetGridZoomable) SetZoom(zoom float64) {
	if zoom < 0.2 {
		zoom = 0.2
	}
	if zoom > 4.0 {
		zoom = 4.0
	}
	g.Zoom = zoom
}

func (g *TilesetGridZoomable) Pan(dx, dy int) {
	g.PanX += dx
	g.PanY += dy
}

func makeOutlineNineSlice(c color.Color) *euiimage.NineSlice {
	img := ebiten.NewImage(3, 3)
	transparent := color.RGBA{0, 0, 0, 0}
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			img.Set(x, y, transparent)
		}
	}
	for i := 0; i < 3; i++ {
		img.Set(i, 0, c)
		img.Set(i, 2, c)
		img.Set(0, i, c)
		img.Set(2, i, c)
	}
	return euiimage.NewNineSliceSimple(img, 1, 1)
}

type TilesetGrid struct {
	Tileset  *ebiten.Image
	TileSize int
	Selected int
}

// NewTilesetGrid creates a widget for displaying the tileset as a grid of selectable tiles.
func NewTilesetGrid(tileset *ebiten.Image, tileSize int, onSelect func(tileIndex int)) *widget.Container {
	if tileset == nil {
		return widget.NewContainer()
	}
	w, h := tileset.Size()
	tilesX := w / tileSize
	tilesY := h / tileSize
	container := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewGridLayout(
				widget.GridLayoutOpts.Columns(tilesX),
				widget.GridLayoutOpts.Spacing(2, 2),
			),
		),
	)
	tileIndex := 0
	for y := 0; y < tilesY; y++ {
		for x := 0; x < tilesX; x++ {
			sub := tileset.SubImage(
				image.Rect(x*tileSize, y*tileSize, (x+1)*tileSize, (y+1)*tileSize),
			).(*ebiten.Image)
			idx := tileIndex
			imgWidget := widget.NewGraphic(
				widget.GraphicOpts.Image(sub),
				widget.GraphicOpts.WidgetOpts(
					widget.WidgetOpts.MinSize(tileSize, tileSize),
					widget.WidgetOpts.MouseButtonClickedHandler(func(args *widget.WidgetMouseButtonClickedEventArgs) {
						onSelect(idx)
					}),
				),
			)
			container.AddChild(imgWidget)
			tileIndex++
		}
	}
	return container
}
