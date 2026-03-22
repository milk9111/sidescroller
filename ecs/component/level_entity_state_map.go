package component

type PersistedLevelEntityState string

const (
	PersistedLevelEntityStateActive    PersistedLevelEntityState = "active"
	PersistedLevelEntityStateDefeated  PersistedLevelEntityState = "defeated"
	PersistedLevelEntityStateCollected PersistedLevelEntityState = "collected"
	PersistedLevelEntityStateUsed      PersistedLevelEntityState = "used"
)

// LevelEntityStateMap stores per-level entity state on the player so the data
// survives level transitions with the player singleton.
type LevelEntityStateMap struct {
	States map[string]PersistedLevelEntityState
}

var LevelEntityStateMapComponent = NewComponent[LevelEntityStateMap]()
