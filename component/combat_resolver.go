package component

import "github.com/milk9111/sidescroller/common"

type hitKey struct {
	HitboxID string
	OwnerID  int
	TargetID int
}

// CombatResolver applies damage between hitboxes and hurtboxes.
type CombatResolver struct {
	Emitter *CombatEventEmitter

	frame    int
	lastHits map[hitKey]int
	// Recent collisions recorded during Resolve()
	Recent []CollisionRecord
}

// NewCombatResolver creates a resolver instance.
func NewCombatResolver() *CombatResolver {
	return &CombatResolver{
		lastHits: make(map[hitKey]int),
	}
}

// Tick advances internal frame counters (call once per game frame).
func (r *CombatResolver) Tick() {
	if r == nil {
		return
	}
	r.frame++
}

// Resolve applies combat between a single damage dealer and hurtbox owner.
// Returns true if any damage was applied.
func (r *CombatResolver) Resolve(dealer DamageDealerComponent, target HurtboxComponent, health HealthComponent) bool {
	if r == nil || dealer == nil || target == nil || health == nil {
		return false
	}
	if !target.CanBeHit() || !health.IsAlive() {
		return false
	}

	dealing := dealer.Hitboxes()
	receiving := target.Hurtboxes()
	if len(dealing) == 0 || len(receiving) == 0 {
		return false
	}

	applied := false
	for _, hb := range dealing {
		if !hb.Active {
			continue
		}
		for _, hu := range receiving {
			if !hu.Enabled {
				continue
			}
			if hb.OwnerID == hu.OwnerID {
				continue
			}
			if !factionCanHit(hb.Damage.Faction, hu.Faction) {
				continue
			}
			if !rectIntersects(&hb.Rect, &hu.Rect) {
				continue
			}

			evt := CombatEvent{
				Type:       EventHit,
				AttackerID: hb.OwnerID,
				TargetID:   hu.OwnerID,
				Damage:     hb.Damage.Amount,
				HitboxID:   hb.ID,
				Frame:      r.frame,
				PosX:       hu.Rect.X + hu.Rect.Width/2,
				PosY:       hu.Rect.Y + hu.Rect.Height/2,
				KnockbackX: hb.Damage.KnockbackX,
				KnockbackY: hb.Damage.KnockbackY,
			}
			if r.Emitter != nil {
				r.Emitter.Emit(evt)
			}

			if r.isOnCooldown(hb, hu) {
				continue
			}

			if health.ApplyDamage(hb.Damage.Amount, evt) {
				applied = true
				r.markHit(hb, hu)
				// record collision rects for debug highlighting
				r.Recent = append(r.Recent, CollisionRecord{Hit: hb.Rect, Hurt: hu.Rect, FramesLeft: 6})
				if hb.Damage.IFrameFrames > 0 {
					health.StartIFrames(hb.Damage.IFrameFrames)
				}
				if r.Emitter != nil {
					evt.Type = EventDamageApplied
					r.Emitter.Emit(evt)
					if !health.IsAlive() {
						evt.Type = EventDeath
						r.Emitter.Emit(evt)
					}
				}
				if !hb.Damage.MultiHit {
					break
				}
			}
		}
	}
	return applied
}

// CollisionRecord stores a recent collision pair for debug highlighting.
type CollisionRecord struct {
	Hit        common.Rect
	Hurt       common.Rect
	FramesLeft int
}

// Global highlights (simple shared store used by debug drawing).
var recentHighlights []CollisionRecord

// AddRecentHighlights appends records to the global highlight store.
func AddRecentHighlights(records []CollisionRecord) {
	if len(records) == 0 {
		return
	}
	recentHighlights = append(recentHighlights, records...)
}

// TickHighlights advances and expires recent highlight records. Call once per frame.
func TickHighlights() {
	if len(recentHighlights) == 0 {
		return
	}
	out := recentHighlights[:0]
	for _, r := range recentHighlights {
		r.FramesLeft--
		if r.FramesLeft > 0 {
			out = append(out, r)
		}
	}
	recentHighlights = out
}

// GetRecentHighlights returns a copy of current recent highlights.
func GetRecentHighlights() []CollisionRecord {
	if len(recentHighlights) == 0 {
		return nil
	}
	cp := make([]CollisionRecord, len(recentHighlights))
	copy(cp, recentHighlights)
	return cp
}

// ResolveAll applies combat for multiple dealers and targets.
// healthByOwner maps owner IDs to health components.
func (r *CombatResolver) ResolveAll(dealers []DamageDealerComponent, targets []HurtboxComponent, healthByOwner map[int]HealthComponent) {
	if r == nil || len(dealers) == 0 || len(targets) == 0 || len(healthByOwner) == 0 {
		return
	}
	for _, t := range targets {
		if t == nil {
			continue
		}
		ownerID := 0
		boxes := t.Hurtboxes()
		if len(boxes) > 0 {
			ownerID = boxes[0].OwnerID
		}
		health := healthByOwner[ownerID]
		if health == nil {
			continue
		}
		for _, d := range dealers {
			if d == nil {
				continue
			}
			r.Resolve(d, t, health)
		}
	}
}

func (r *CombatResolver) isOnCooldown(hb Hitbox, hu Hurtbox) bool {
	if hb.Damage.MultiHit {
		return false
	}
	if hb.Damage.CooldownFrames <= 0 {
		return false
	}
	if r.lastHits == nil {
		r.lastHits = make(map[hitKey]int)
	}
	key := hitKey{HitboxID: hb.ID, OwnerID: hb.OwnerID, TargetID: hu.OwnerID}
	last, ok := r.lastHits[key]
	if !ok {
		return false
	}
	return (r.frame - last) < hb.Damage.CooldownFrames
}

func (r *CombatResolver) markHit(hb Hitbox, hu Hurtbox) {
	if r.lastHits == nil {
		r.lastHits = make(map[hitKey]int)
	}
	key := hitKey{HitboxID: hb.ID, OwnerID: hb.OwnerID, TargetID: hu.OwnerID}
	r.lastHits[key] = r.frame
}

func factionCanHit(attacker Faction, target Faction) bool {
	if attacker == FactionNeutral || target == FactionNeutral {
		return true
	}
	return attacker != target
}

func rectIntersects(a, b *common.Rect) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Intersects(b)
}
