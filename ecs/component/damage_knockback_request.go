package component

// DamageKnockback is a transient component requesting the DamageKnockback
// system apply an impulse to the entity. Systems should add this component
// and the DamageKnockbackSystem will perform the physics impulse then
// remove the component.
type DamageKnockback struct {
	SourceX      float64
	SourceY      float64
	Strong       bool
	SourceEntity uint64
}

var DamageKnockbackRequestComponent = NewComponent[DamageKnockback]()
