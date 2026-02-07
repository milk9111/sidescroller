package systems

import (
	"math"
	"time"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
)

// PickupSystem updates pickup float and applies effects on overlap.
type PickupSystem struct{}

// NewPickupSystem creates a PickupSystem.
func NewPickupSystem() *PickupSystem {
	return &PickupSystem{}
}

// Update floats pickups and applies ability unlocks.
func (s *PickupSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	pickups := w.Pickups()
	if pickups == nil {
		return
	}

	player := ecs.Entity{ID: 0, Gen: 0}
	if pset := w.PlayerControllers(); pset != nil && len(pset.Entities()) > 0 {
		player.ID = pset.Entities()[0]
	}
	if player.ID == 0 {
		return
	}
	ptr := w.GetTransform(player)
	pcol := w.GetCollider(player)
	pctrl := w.GetPlayerController(player)
	if ptr == nil || pcol == nil || pctrl == nil {
		return
	}
	playerRect := common.Rect{X: ptr.X, Y: ptr.Y, Width: pcol.Width, Height: pcol.Height}

	now := float64(time.Now().UnixNano()) / 1e9
	for _, id := range pickups.Entities() {
		pv := pickups.Get(id)
		pk, ok := pv.(*components.Pickup)
		if !ok || pk == nil || !pk.Enabled {
			continue
		}
		ent := ecs.Entity{ID: id, Gen: 0}
		tr := w.GetTransform(ent)
		if tr == nil {
			continue
		}
		amp := float64(pk.Amplitude)
		freq := float64(pk.Frequency)
		if freq == 0 {
			freq = 2.0
		}
		yOffset := float32(math.Sin(now*freq+pk.Phase) * amp)
		tr.X = pk.BaseX
		tr.Y = pk.BaseY + yOffset

		col := w.GetCollider(ent)
		if col == nil {
			col = &components.Collider{Width: pk.Width, Height: pk.Height}
			w.SetCollider(ent, col)
		}
		pickupRect := common.Rect{X: tr.X, Y: tr.Y, Width: col.Width, Height: col.Height}
		if !pickupRect.Intersects(&playerRect) {
			continue
		}

		switch pk.Kind {
		case "double_jump":
			pctrl.DoubleJump = true
			if pctrl.MaxJumps < 2 {
				pctrl.MaxJumps = 2
			}
		case "dash":
			pctrl.Dash = true
		case "anchor":
			pctrl.Swing = true
		}

		pk.Enabled = false
		w.Pickups().Remove(id)
		w.Transforms().Remove(id)
		w.Sprites().Remove(id)
		w.Colliders().Remove(id)
		w.DestroyEntity(ent)
	}
}
