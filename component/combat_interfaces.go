package component

// HealthComponent exposes health operations for combat systems.
type HealthComponent interface {
	IsAlive() bool
	ApplyDamage(amount float32, evt CombatEvent) bool
	Heal(amount float32)
	StartIFrames(frames int)
	Tick()
	CurrentHP() float32
	MaxHP() float32
	SetCurrentHP(v float32)
	SetMaxHP(v float32)
}

// DamageDealerComponent exposes hitbox data and event emission.
type DamageDealerComponent interface {
	Hitboxes() []Hitbox
	SetHitboxes(boxes []Hitbox)
	DamageFaction() Faction
	EmitHit(evt CombatEvent)
}

// HurtboxComponent exposes defensive collision data.
type HurtboxComponent interface {
	Hurtboxes() []Hurtbox
	SetHurtboxes(boxes []Hurtbox)
	HurtboxFaction() Faction
	CanBeHit() bool
}
