package component

import "image/color"

// LineRender defines a world-space line to render.
type LineRender struct {
	StartX    float64
	StartY    float64
	EndX      float64
	EndY      float64
	Width     float32
	Color     color.Color
	AntiAlias bool
}

var LineRenderComponent = NewComponent[LineRender]()
