package component

// StaticTileBatchState is a small component stored on a level-bound entity that
// indicates whether the static tile batch needs to be rebuilt. Systems should
// set `Dirty = true` when static tile related state changes.
type StaticTileBatchState struct {
	Dirty bool
}

var StaticTileBatchStateComponent = NewComponent[StaticTileBatchState]()
