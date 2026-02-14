package component

// ScreenSpace marks renderable entities that should be drawn in screen/UI space
// (not affected by camera translation or zoom).
type ScreenSpace struct{}

var ScreenSpaceComponent = NewComponent[ScreenSpace]()
