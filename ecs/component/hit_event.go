package component

// HitEvent is a transient marker added to an attacker when their hitbox
// successfully damages a target. Systems may consume this to trigger
// one-shot responses (SFX, particle spawn) in the attacker's context.
type HitEvent struct{}

var HitEventComponent = NewComponent[HitEvent]()
