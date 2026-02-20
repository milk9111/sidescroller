package component

// Knockbackable marks entities that may receive knockback from damage.
// Only entities with this component will be affected by the DamageKnockbackSystem.
type Knockbackable struct{}

var KnockbackableComponent = NewComponent[Knockbackable]()
