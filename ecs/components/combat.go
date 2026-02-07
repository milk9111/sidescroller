package components

import "github.com/milk9111/sidescroller/component"

// DamageDealer stores hitboxes and combat event emission.
type DamageDealer struct {
	Boxes   []component.Hitbox
	Faction component.Faction
	Emitter *component.CombatEventEmitter
}

// Hitboxes returns the current hitboxes.
func (d *DamageDealer) Hitboxes() []component.Hitbox {
	if d == nil {
		return nil
	}
	return d.Boxes
}

// SetHitboxes updates hitboxes.
func (d *DamageDealer) SetHitboxes(boxes []component.Hitbox) {
	if d == nil {
		return
	}
	d.Boxes = boxes
}

// DamageFaction returns the faction for the dealer.
func (d *DamageDealer) DamageFaction() component.Faction {
	if d == nil {
		return component.FactionNeutral
	}
	return d.Faction
}

// EmitHit emits a combat event if an emitter is set.
func (d *DamageDealer) EmitHit(evt component.CombatEvent) {
	if d == nil || d.Emitter == nil {
		return
	}
	d.Emitter.Emit(evt)
}

// HurtboxSet stores hurtboxes and hit filtering.
type HurtboxSet struct {
	Boxes   []component.Hurtbox
	Faction component.Faction
	Enabled bool
}

// Hurtboxes returns the current hurtboxes.
func (h *HurtboxSet) Hurtboxes() []component.Hurtbox {
	if h == nil {
		return nil
	}
	return h.Boxes
}

// SetHurtboxes updates hurtboxes.
func (h *HurtboxSet) SetHurtboxes(boxes []component.Hurtbox) {
	if h == nil {
		return
	}
	h.Boxes = boxes
}

// HurtboxFaction returns the faction for the hurtboxes.
func (h *HurtboxSet) HurtboxFaction() component.Faction {
	if h == nil {
		return component.FactionNeutral
	}
	return h.Faction
}

// CanBeHit reports whether the hurtboxes can be hit.
func (h *HurtboxSet) CanBeHit() bool {
	if h == nil {
		return false
	}
	if len(h.Boxes) == 0 {
		return false
	}
	if !h.Enabled {
		return false
	}
	return true
}
