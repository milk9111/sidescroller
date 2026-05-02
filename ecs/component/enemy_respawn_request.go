package component

// EnemyRespawnRequest asks the persistence system to reload the current level
// from the player's current snapshot while clearing defeated-enemy state for
// the active level.
type EnemyRespawnRequest struct{}

var EnemyRespawnRequestComponent = NewComponent[EnemyRespawnRequest]()
