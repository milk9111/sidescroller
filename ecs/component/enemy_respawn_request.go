package component

// EnemyRespawnRequest asks the persistence system to rebuild authored enemy
// entities for the active level from level data while preserving the rest of
// the current world.
type EnemyRespawnRequest struct{}

var EnemyRespawnRequestComponent = NewComponent[EnemyRespawnRequest]()
