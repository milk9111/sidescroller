package component

// TransitionCooldown prevents immediately re-triggering a transition when the
// player spawns into a linked transition area. The TransitionSystem will only
// allow activation once the player has fully left the cooldown transition area.
type TransitionCooldown struct {
	Active       bool
	TransitionID string
}

var TransitionCooldownComponent = NewComponent[TransitionCooldown]()
