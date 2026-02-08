package component

// RenderLayer is used to sort draw order deterministically.
type RenderLayer struct {
	Index int
}

var RenderLayerComponent = NewComponent[RenderLayer]()
