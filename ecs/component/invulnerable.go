package component

// Invulnerable marks an entity as temporarily immune to damage.
type Invulnerable struct{}

var InvulnerableComponent = NewComponent[Invulnerable]()
