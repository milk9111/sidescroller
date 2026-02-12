package main

import "github.com/ebitenui/ebitenui/widget"

func buildGridPanel() *widget.Container {
	// Main grid container (placeholder). The editor's canvas is drawn by Ebiten;
	// this container exists primarily so the root anchor layout has a center child.
	return widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(800, 600),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
}
