package systems

import (
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// PlayerCombatSystem updates player hitboxes and hurtboxes.
type PlayerCombatSystem struct{}

// NewPlayerCombatSystem creates a PlayerCombatSystem.
func NewPlayerCombatSystem() *PlayerCombatSystem {
	return &PlayerCombatSystem{}
}

// Update refreshes hurtboxes and manages attack hitboxes.
func (s *PlayerCombatSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	set := w.PlayerControllers()
	if set == nil {
		return
	}

	for _, id := range set.Entities() {
		pv := set.Get(id)
		ctrl, ok := pv.(*components.PlayerController)
		if !ok || ctrl == nil {
			continue
		}
		ent := ecs.Entity{ID: id, Gen: 0}
		tr := w.GetTransform(ent)
		col := w.GetCollider(ent)
		if tr == nil || col == nil {
			continue
		}

		input := w.GetInput(ent)
		if input != nil && input.MouseLeftPressed && ctrl.AttackTimer <= 0 {
			ctrl.AttackTimer = ctrl.AttackFrames
		}

		if anim := w.GetAnimator(ent); anim != nil {
			if ctrl.AttackTimer > 0 && ctrl.AttackAnim != nil && anim.Anim != ctrl.AttackAnim {
				anim.Anim = ctrl.AttackAnim
				anim.Anim.Reset()
			}
		}

		hurt := w.GetHurtbox(ent)
		if hurt == nil {
			hurt = &components.HurtboxSet{Faction: component.FactionPlayer, Enabled: true}
			w.SetHurtbox(ent, hurt)
		}
		if len(hurt.Boxes) == 0 {
			hurt.Boxes = []component.Hurtbox{{
				ID:      "player_body",
				OwnerID: id,
				Faction: component.FactionPlayer,
				Enabled: true,
			}}
		}
		for i := range hurt.Boxes {
			h := hurt.Boxes[i]
			h.Rect = common.Rect{X: tr.X, Y: tr.Y, Width: col.Width, Height: col.Height}
			h.OwnerID = id
			h.Enabled = true
			h.Faction = component.FactionPlayer
			hurt.Boxes[i] = h
		}

		dealer := w.GetDamageDealer(ent)
		if dealer == nil {
			dealer = &components.DamageDealer{Faction: component.FactionPlayer}
			w.SetDamageDealer(ent, dealer)
		}

		if ctrl.AttackTimer > 0 {
			ctrl.AttackTimer--
			hbW := float32(28)
			hbH := col.Height * 0.8
			var hbX float32
			if ctrl.FacingRight {
				hbX = tr.X + col.Width
			} else {
				hbX = tr.X - hbW
			}
			hbY := tr.Y + (col.Height-hbH)/2
			hb := component.Hitbox{
				ID:      "player_attack",
				OwnerID: id,
				Active:  true,
				Rect:    common.Rect{X: hbX, Y: hbY, Width: hbW, Height: hbH},
				Damage: component.Damage{
					Amount:         1,
					KnockbackX:     2.0,
					KnockbackY:     -1.0,
					HitstunFrames:  6,
					CooldownFrames: 12,
					IFrameFrames:   8,
					Faction:        component.FactionPlayer,
					MultiHit:       false,
				},
			}
			dealer.Boxes = []component.Hitbox{hb}
		} else {
			dealer.Boxes = nil
		}
	}
}
