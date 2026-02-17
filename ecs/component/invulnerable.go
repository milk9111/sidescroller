package component

// Invulnerable marks an entity as temporarily immune to damage.
// If Frames > 0 the system will count frames down each tick and remove the
// component when it reaches zero. Frames == 0 means indefinite invulnerability
// until explicitly removed.
type Invulnerable struct {
	Frames int
}

var InvulnerableComponent = NewComponent[Invulnerable]()
