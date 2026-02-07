package systems

import (
	"math"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/obj"
)

const (
	bulletDamageAmount   = 1
	bulletKnockbackX     = 1.5
	bulletKnockbackY     = -0.5
	bulletIFrameFrames   = 10
	bulletCooldownFrames = 10
)

// BulletSystem updates bullets and handles collisions.
type BulletSystem struct {
	Level *obj.Level
}

// NewBulletSystem creates a BulletSystem.
func NewBulletSystem(level *obj.Level) *BulletSystem {
	return &BulletSystem{Level: level}
}

// Update moves bullets and removes them when they expire or hit solid tiles.
func (s *BulletSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	bullets := w.Bullets()
	if bullets == nil {
		return
	}
	trs := w.Transforms()
	vels := w.Velocities()
	dealers := w.DamageDealers()
	if trs == nil || vels == nil || dealers == nil {
		return
	}

	for _, id := range bullets.Entities() {
		bv := bullets.Get(id)
		b, ok := bv.(*components.Bullet)
		if !ok || b == nil {
			continue
		}
		trv := trs.Get(id)
		tr, ok := trv.(*components.Transform)
		if !ok || tr == nil {
			continue
		}
		vv := vels.Get(id)
		vel, ok := vv.(*components.Velocity)
		if !ok || vel == nil {
			continue
		}

		tr.X += vel.VX
		tr.Y += vel.VY
		b.AgeFrames++

		if b.LifeFrames > 0 && b.AgeFrames >= b.LifeFrames {
			despawnBullet(w, id)
			continue
		}

		if s.hitSolid(tr.X, tr.Y, b.Width, b.Height) {
			despawnBullet(w, id)
			continue
		}

		dv := dealers.Get(id)
		dealer, ok := dv.(*components.DamageDealer)
		if !ok || dealer == nil {
			dealer = &components.DamageDealer{Faction: component.FactionEnemy}
			dealers.Set(id, dealer)
		}
		hb := component.Hitbox{
			ID:      "bullet",
			OwnerID: id,
			Active:  true,
			Rect:    common.Rect{X: tr.X, Y: tr.Y, Width: b.Width, Height: b.Height},
			Damage:  b.Damage,
		}
		if hb.Damage.Amount == 0 {
			hb.Damage = component.Damage{
				Amount:         bulletDamageAmount,
				KnockbackX:     bulletKnockbackX,
				KnockbackY:     bulletKnockbackY,
				HitstunFrames:  6,
				CooldownFrames: bulletCooldownFrames,
				IFrameFrames:   bulletIFrameFrames,
				Faction:        component.FactionEnemy,
				MultiHit:       false,
			}
		}
		dealer.Boxes = []component.Hitbox{hb}
	}
}

// SpawnBullet creates a bullet entity.
func SpawnBullet(w *ecs.World, x, y, vx, vy float32, ownerID int) ecs.Entity {
	if w == nil {
		return ecs.Entity{}
	}
	e := w.CreateEntity()
	w.SetTransform(e, &components.Transform{X: x, Y: y})
	w.SetVelocity(e, &components.Velocity{VX: vx, VY: vy})
	w.SetSprite(e, &components.Sprite{ImageKey: "flying_enemy_bullet.png", Width: 32, Height: 32, Layer: 2})
	w.SetBullet(e, &components.Bullet{OwnerID: ownerID, Width: 32, Height: 32})
	w.SetDamageDealer(e, &components.DamageDealer{Faction: component.FactionEnemy})
	return e
}

func (s *BulletSystem) hitSolid(x, y, wth, hgt float32) bool {
	if s == nil || s.Level == nil {
		return false
	}
	level := s.Level
	left := int(math.Floor(float64(x) / float64(common.TileSize)))
	top := int(math.Floor(float64(y) / float64(common.TileSize)))
	right := int(math.Floor(float64(x+wth-1) / float64(common.TileSize)))
	bottom := int(math.Floor(float64(y+hgt-1) / float64(common.TileSize)))
	if right < 0 || bottom < 0 || left >= level.Width || top >= level.Height {
		return true
	}
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if right >= level.Width {
		right = level.Width - 1
	}
	if bottom >= level.Height {
		bottom = level.Height - 1
	}
	for yy := top; yy <= bottom; yy++ {
		for xx := left; xx <= right; xx++ {
			if level.PhysicsLayers != nil && len(level.PhysicsLayers) > 0 {
				for _, ly := range level.PhysicsLayers {
					if ly == nil || ly.Tiles == nil || len(ly.Tiles) != level.Width*level.Height {
						continue
					}
					if ly.Tiles[yy*level.Width+xx] != 0 {
						return true
					}
				}
			} else if isPhysicsTile(level, xx, yy) {
				return true
			}
		}
	}
	return false
}

func isPhysicsTile(level *obj.Level, x, y int) bool {
	if level == nil {
		return false
	}
	if x < 0 || y < 0 || x >= level.Width || y >= level.Height {
		return true
	}
	if level.Layers == nil || len(level.Layers) == 0 {
		return false
	}
	idx := y*level.Width + x
	for layerIdx, layer := range level.Layers {
		if layer == nil || len(layer) != level.Width*level.Height {
			continue
		}
		if level.LayerMeta == nil || layerIdx >= len(level.LayerMeta) || !level.LayerMeta[layerIdx].HasPhysics {
			continue
		}
		if layer[idx] != 0 {
			return true
		}
	}
	return false
}

func despawnBullet(w *ecs.World, id int) {
	if w == nil {
		return
	}
	w.Bullets().Remove(id)
	w.Transforms().Remove(id)
	w.Velocities().Remove(id)
	w.Sprites().Remove(id)
	w.DamageDealers().Remove(id)
	w.DestroyEntity(ecs.Entity{ID: id, Gen: 0})
}
