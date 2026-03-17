package component

// MergedLevelPhysics marks generated static colliders built from the union of active physics layers.
type MergedLevelPhysics struct{}

var MergedLevelPhysicsComponent = NewComponent[MergedLevelPhysics]()
