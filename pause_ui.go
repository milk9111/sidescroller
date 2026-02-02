package main

import (
	"image/color"

	"github.com/milk9111/sidescroller/common"
	"golang.org/x/image/font/basicfont"

	"github.com/ebitenui/ebitenui"
	imageui "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	ebtext "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// NewPauseUI builds a simple centered pause menu with Resume and Quit buttons.
// This creates buttons using colored nine-slices and no embedded text faces,
// so it doesn't require theme fonts to be loaded.
func NewPauseUI(g *Game) *ebitenui.UI {
	// semi-transparent panel background
	panelImg := imageui.NewNineSliceColor(color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 200})
	// simple button color
	btnImg := imageui.NewNineSliceColor(color.NRGBA{R: 0x33, G: 0x33, B: 0x33, A: 255})

	// Create a text.Face from the built-in basic font so we can show labels
	goFace := ebtext.NewGoXFace(basicfont.Face7x13)
	var face ebtext.Face = goFace

	// Button text color
	btnTextColor := &widget.ButtonTextColor{Idle: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}}

	// Title text
	title := widget.NewText(
		widget.TextOpts.Text("Paused", &face, color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)

	// Resume button
	resumeBtn := widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{Idle: btnImg, Pressed: btnImg}),
		widget.ButtonOpts.Text("Resume", &face, btnTextColor),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			g.paused = false
		}),
	)

	// Panel with vertical layout. Set MinSize to ~50% of base resolution and center in anchor layout.
	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(panelImg),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(&widget.Insets{Top: 20, Bottom: 20, Left: 30, Right: 30}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(common.BaseWidth/2, common.BaseHeight/2),
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter, VerticalPosition: widget.AnchorLayoutPositionCenter}),
		),
	)
	panel.AddChild(title)
	panel.AddChild(resumeBtn)

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// center the panel in the root
	root.AddChild(panel)

	ui := &ebitenui.UI{Container: root}
	return ui
}
