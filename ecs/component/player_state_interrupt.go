package component

// PlayerStateInterrupt is used as a one-shot event to request an immediate
// state change on the player controller system. Systems should add this
// component to a player entity and the controller will consume it.
type PlayerStateInterrupt struct {
	State string
}

var PlayerStateInterruptComponent = NewComponent[PlayerStateInterrupt]()
