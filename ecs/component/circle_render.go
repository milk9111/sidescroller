package component

import "image/color"

// CircleRender defines a stroked circle to render in world-space or screen-space.
type CircleRender struct {
	OffsetX   float64
	OffsetY   float64
	Radius    float64
	Width     float32
	Color     color.Color
	AntiAlias bool
	Disabled  bool
}

var CircleRenderComponent = NewComponent[CircleRender]()
