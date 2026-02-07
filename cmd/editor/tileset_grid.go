package main

import (
	"image"

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
	g.Container = widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewAnchorLayout(),
		),
	)
	// Add a single graphic for the tileset
	graphic := widget.NewGraphic(
		widget.GraphicOpts.Image(tileset),
		widget.GraphicOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(240, 240),
			widget.WidgetOpts.MouseButtonClickedHandler(func(args *widget.WidgetMouseButtonClickedEventArgs) {
				// TODO: handle tile selection
			}),
		),
	)
	g.Container.AddChild(graphic)
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
