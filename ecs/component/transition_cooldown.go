package component

// TransitionCooldown prevents immediately re-triggering a transition when the
// player spawns into a linked transition area. The TransitionSystem will only
// allow activation once the player has fully left every cooldown transition
// area they initially overlapped.
type TransitionCooldown struct {
	Active        bool
	TransitionID  string
	TransitionIDs []string
}

var TransitionCooldownComponent = NewComponent[TransitionCooldown]()
