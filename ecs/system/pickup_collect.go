package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type PickupCollectSystem struct{}

func NewPickupCollectSystem() *PickupCollectSystem { return &PickupCollectSystem{} }

func (s *PickupCollectSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	playerTransform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || playerTransform == nil {
		return
	}

	playerBody, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind())
	if !ok || playerBody == nil {
		return
	}

	px := aabbTopLeftX(w, player, playerTransform.X, playerBody.OffsetX, playerBody.Width, playerBody.AlignTopLeft)
	py := playerTransform.Y + playerBody.OffsetY
	if !playerBody.AlignTopLeft {
		py -= playerBody.Height / 2
	}
	pw := playerBody.Width
	ph := playerBody.Height

	ecs.ForEach2(w, component.PickupComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, pickup *component.Pickup, t *component.Transform) {
		if pickup == nil || t == nil {
			return
		}

		kw := pickup.CollisionWidth
		kh := pickup.CollisionHeight
		if kw <= 0 || kh <= 0 {
			kw = 24
			kh = 24
		}

		kx := t.X
		ky := t.Y
		if !intersects(px, py, pw, ph, kx, ky, kw, kh) {
			return
		}

		if audioComp, ok := ecs.Get(w, e, component.AudioComponent.Kind()); ok && audioComp != nil {
			for i, name := range audioComp.Names {
				if name != "pickup" {
					continue
				}
				if i < len(audioComp.Play) {
					audioComp.Play[i] = true
				}
				break
			}
			_ = ecs.Add(w, e, component.AudioComponent.Kind(), audioComp)
		}

		if abilitiesEntity, found := ecs.First(w, component.AbilitiesComponent.Kind()); found {
			if abilities, ok := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind()); ok && abilities != nil {
				if pickup.GrantDoubleJump {
					abilities.DoubleJump = true
				}
				if pickup.GrantWallGrab {
					abilities.WallGrab = true
				}
				if pickup.GrantAnchor {
					abilities.Anchor = true
				}
				_ = ecs.Add(w, abilitiesEntity, component.AbilitiesComponent.Kind(), abilities)
			}
		} else {
			ent := ecs.CreateEntity(w)
			_ = ecs.Add(w, ent, component.AbilitiesComponent.Kind(), &component.Abilities{
				DoubleJump: pickup.GrantDoubleJump,
				WallGrab:   pickup.GrantWallGrab,
				Anchor:     pickup.GrantAnchor,
			})
		}

		if pickup.Kind == "trophy" {
			if counterEntity, found := ecs.First(w, component.TrophyCounterComponent.Kind()); found {
				if counter, ok := ecs.Get(w, counterEntity, component.TrophyCounterComponent.Kind()); ok && counter != nil {
					counter.Collected++
					if counter.Total >= 0 && counter.Collected > counter.Total {
						counter.Collected = counter.Total
					}
					_ = ecs.Add(w, counterEntity, component.TrophyCounterComponent.Kind(), counter)
				}
			}
		}

		// AudioSystem runs before PickupCollectSystem in the scheduler. If we
		// destroy immediately, queued pickup audio never gets processed.
		// Remove pickup behavior now, hide sprite, and destroy shortly after.
		_ = ecs.Remove(w, e, component.PickupComponent.Kind())
		_ = ecs.Remove(w, e, component.SpriteComponent.Kind())
		_ = ecs.Add(w, e, component.TTLComponent.Kind(), &component.TTL{Frames: 2})
	})
}
