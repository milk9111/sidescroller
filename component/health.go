package component

// Health is a reusable health component for any entity that can take damage.
type Health struct {
	Max     float32
	Current float32
	IFrames int
	Dead    bool

	OnDamage      func(h *Health, evt CombatEvent)
	OnDeath       func(h *Health, evt CombatEvent)
	OnIFrameStart func(h *Health)
	OnIFrameEnd   func(h *Health)
}

// NewHealth creates a Health component with max/current initialized.
func NewHealth(max float32) *Health {
	if max <= 0 {
		max = 1
	}
	return &Health{Max: max, Current: max}
}

// IsAlive reports whether the entity is alive.
func (h *Health) IsAlive() bool {
	return h != nil && !h.Dead && h.Current > 0
}

// ApplyDamage applies damage if not in i-frames. Returns true if damage was applied.
func (h *Health) ApplyDamage(amount float32, evt CombatEvent) bool {
	if h == nil || h.Dead || h.IFrames > 0 || amount <= 0 {
		return false
	}
	h.Current -= amount
	if h.Current < 0 {
		h.Current = 0
	}
	if h.OnDamage != nil {
		h.OnDamage(h, evt)
	}
	if h.Current <= 0 {
		h.Dead = true
		if h.OnDeath != nil {
			h.OnDeath(h, evt)
		}
	}
	return true
}

// Heal restores health up to Max.
func (h *Health) Heal(amount float32) {
	if h == nil || h.Dead || amount <= 0 {
		return
	}
	h.Current += amount
	if h.Current > h.Max {
		h.Current = h.Max
	}
}

// StartIFrames sets invulnerability frames.
func (h *Health) StartIFrames(frames int) {
	if h == nil || frames <= 0 {
		return
	}
	if h.IFrames <= 0 && h.OnIFrameStart != nil {
		h.OnIFrameStart(h)
	}
	h.IFrames = frames
}

// Tick advances the i-frame timer by one frame.
func (h *Health) Tick() {
	if h == nil || h.IFrames <= 0 {
		return
	}
	h.IFrames--
	if h.IFrames <= 0 {
		h.IFrames = 0
		if h.OnIFrameEnd != nil {
			h.OnIFrameEnd(h)
		}
	}
}

// CurrentHP returns the current health value.
func (h *Health) CurrentHP() float32 {
	if h == nil {
		return 0
	}
	return h.Current
}

// MaxHP returns the maximum health value.
func (h *Health) MaxHP() float32 {
	if h == nil {
		return 0
	}
	return h.Max
}

// SetCurrentHP sets the current health value and clamps to [0, Max].
func (h *Health) SetCurrentHP(v float32) {
	if h == nil {
		return
	}
	h.Current = v
	if h.Current < 0 {
		h.Current = 0
	}
	if h.Max > 0 && h.Current > h.Max {
		h.Current = h.Max
	}
}

// SetMaxHP sets the maximum health value and clamps Current if needed.
func (h *Health) SetMaxHP(v float32) {
	if h == nil {
		return
	}
	h.Max = v
	if h.Max <= 0 {
		h.Max = 1
	}
	if h.Current > h.Max {
		h.Current = h.Max
	}
}
