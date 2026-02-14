package component

// SpriteBlackout marks sprites that should be rendered black while preserving
// their alpha silhouette.
type SpriteBlackout struct{}

var SpriteBlackoutComponent = NewComponent[SpriteBlackout]()
