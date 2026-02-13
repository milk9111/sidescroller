package component

// StaticTile marks a world tile that can be pre-batched into cached chunk images.
type StaticTile struct{}

var StaticTileComponent = NewComponent[StaticTile]()
