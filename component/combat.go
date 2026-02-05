package component

import "github.com/milk9111/sidescroller/common"

// Faction identifies teams for friendly-fire checks.
type Faction int

const (
	FactionNeutral Faction = iota
	FactionPlayer
	FactionEnemy
	FactionEnvironment
)

// CombatEventType defines the kind of combat event.
type CombatEventType string

const (
	EventHit           CombatEventType = "hit"
	EventDamageApplied CombatEventType = "damage_applied"
	EventDeath         CombatEventType = "death"
	EventIFrameStart   CombatEventType = "iframe_start"
	EventIFrameEnd     CombatEventType = "iframe_end"
)

// CombatEvent is emitted during combat resolution.
type CombatEvent struct {
	Type       CombatEventType
	AttackerID int
	TargetID   int
	Damage     float32
	HitboxID   string
	Frame      int
	PosX       float32
	PosY       float32
	KnockbackX float32
	KnockbackY float32
}

// CombatEventHandler handles combat events.
type CombatEventHandler func(evt CombatEvent)

// CombatEventEmitter allows components to emit combat events.
type CombatEventEmitter struct {
	Handlers []CombatEventHandler
}

// Emit sends a combat event to all handlers.
func (e *CombatEventEmitter) Emit(evt CombatEvent) {
	if e == nil || len(e.Handlers) == 0 {
		return
	}
	for _, h := range e.Handlers {
		if h != nil {
			h(evt)
		}
	}
}

// Damage describes damage parameters.
type Damage struct {
	Amount         float32
	KnockbackX     float32
	KnockbackY     float32
	HitstunFrames  int
	CooldownFrames int
	IFrameFrames   int
	Faction        Faction
	MultiHit       bool
}

// Hitbox represents an offensive collision area.
type Hitbox struct {
	ID      string
	Rect    common.Rect
	Damage  Damage
	Active  bool
	OwnerID int
}

// Hurtbox represents a defensive collision area.
type Hurtbox struct {
	ID      string
	Rect    common.Rect
	Faction Faction
	Enabled bool
	OwnerID int
}
