package component

// EntityLayer ties a world entity to a level layer.
type EntityLayer struct {
	Index int
}

var EntityLayerComponent = NewComponent[EntityLayer]()
