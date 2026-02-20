package component

// Cooldown is a simple frame-based cooldown marker. Systems may add this
// component to an entity to start a countdown; when Frames reaches zero the
// component will be removed and interested systems may react (e.g. enqueue an
// event).
type Cooldown struct {
	// Frames remaining for the cooldown (in update ticks)
	Frames int
}

var CooldownComponent = NewComponent[Cooldown]()
