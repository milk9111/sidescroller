package system

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type CombatSystem struct{}

func NewCombatSystem() *CombatSystem { return &CombatSystem{} }

func intersects(ax, ay, aw, ah, bx, by, bw, bh float64) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

func frameActive(frames []int, frame int) bool {
	for _, f := range frames {
		if f == frame {
			return true
		}
	}
	return false
}

func (s *CombatSystem) Update(w *ecs.World) {
	// For each entity that has hitboxes, check configured frames and test against all hurtboxes
	ecs.ForEach3(
		w,
		component.HitboxComponent.Kind(),
		component.TransformComponent.Kind(),
		component.AnimationComponent.Kind(),
		func(e ecs.Entity, hitboxes *[]component.Hitbox, transform *component.Transform, anim *component.Animation) {
			for _, hb := range *hitboxes {
				if hb.Anim == "" || hb.Anim != anim.Current {
					continue
				}
				if len(hb.Frames) > 0 && !frameActive(hb.Frames, anim.Frame) {
					continue
				}

				// Compute hitbox world AABB (flip horizontally when facing left)
				scaleX := transform.ScaleX
				if scaleX == 0 {
					scaleX = 1
				}
				// base offset scaled to world units
				baseOff := hb.OffsetX * scaleX
				offX := baseOff
				if s, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && s.FacingLeft {
					// try to mirror around the sprite frame width if available
					if animDef, ok2 := anim.Defs[anim.Current]; ok2 {
						imgW := float64(animDef.FrameW)
						offX = imgW*scaleX - baseOff - hb.Width
					} else {
						offX = -baseOff - hb.Width
					}
				}
				hx := transform.X + offX
				hy := transform.Y + hb.OffsetY*transform.ScaleY
				hw := hb.Width
				hh := hb.Height

				ecs.ForEach2(w, component.HurtboxComponent.Kind(), component.TransformComponent.Kind(), func(et ecs.Entity, hurtboxes *[]component.Hurtbox, tTransform *component.Transform) {
					if et == e {
						return
					}
					// Don't allow AI (enemies) to damage other AI — skip friendly fire between enemies.
					if ecs.Has(w, e, component.AITagComponent.Kind()) && ecs.Has(w, et, component.AITagComponent.Kind()) {
						return
					}
					for _, hurt := range *hurtboxes {
						tx := tTransform.X + hurt.OffsetX
						ty := tTransform.Y + hurt.OffsetY
						tw := hurt.Width
						th := hurt.Height

						if intersects(hx, hy, hw, hh, tx, ty, tw, th) {
							// Skip if target is temporarily invulnerable
							if ecs.Has(w, et, component.InvulnerableComponent.Kind()) {
								continue
							}
							// Skip if target is already in the death state (no further damage)
							if sm, ok := ecs.Get(w, et, component.PlayerStateMachineComponent.Kind()); ok && sm.State != nil && sm.State.Name() == "death" {
								continue
							}

							// Apply damage if target has health
							if h, ok := ecs.Get(w, et, component.HealthComponent.Kind()); ok {
								h.Current -= hb.Damage
								if h.Current < 0 {
									h.Current = 0
									fmt.Println("Entity", et, "defeated!")
								}
								ecs.Add(w, et, component.HealthComponent.Kind(), h)
								applyDamageKnockback(w, et, hx+hw/2, hy+hh/2)
								// If this target is a player, send a state interrupt requesting
								// either the 'hit' or 'death' state depending on remaining HP.
								if ecs.Has(w, et, component.PlayerTagComponent.Kind()) {
									// Always request the 'hit' state so the hit animation,
									// white flash and SFX play even when this damage
									// reduces HP to zero. The player controller will
									// schedule the subsequent 'death' transition if
									// health is zero after the hit state completes.
									err := ecs.Add(w, et, component.PlayerStateInterruptComponent.Kind(), &component.PlayerStateInterrupt{State: "hit"})
									if err != nil {
										panic("combat: add player state interrupt: " + err.Error())
									}

									shakeFrames := 8
									shakeIntensity := 3.0
									if p, ok := ecs.Get(w, et, component.PlayerComponent.Kind()); ok && p != nil && p.DamageShakeIntensity > 0 {
										shakeIntensity = p.DamageShakeIntensity
									}
									if camEntity, ok := ecs.First(w, component.CameraComponent.Kind()); ok {
										if existing, ok := ecs.Get(w, camEntity, component.CameraShakeRequestComponent.Kind()); ok && existing != nil {
											if existing.Frames > shakeFrames {
												shakeFrames = existing.Frames
											}
											if existing.Intensity > shakeIntensity {
												shakeIntensity = existing.Intensity
											}
										}
										_ = ecs.Add(w, camEntity, component.CameraShakeRequestComponent.Kind(), &component.CameraShakeRequest{Frames: shakeFrames, Intensity: shakeIntensity})
									}
								}
								// If this target is an AI (enemy), request the AI FSM handle a 'hit' event
								if ecs.Has(w, et, component.AITagComponent.Kind()) {
									err := ecs.Add(w, et, component.AIStateInterruptComponent.Kind(), &component.AIStateInterrupt{Event: "hit"})
									if err != nil {
										panic("combat: add ai state interrupt: " + err.Error())
									}
								}
								// If the player dealt damage to an enemy, request a short global hit-freeze.
								if ecs.Has(w, e, component.PlayerTagComponent.Kind()) && ecs.Has(w, et, component.AITagComponent.Kind()) {
									// Determine freeze frames from the attacker's player config if available.
									freezeFrames := 5
									if p, ok := ecs.Get(w, e, component.PlayerComponent.Kind()); ok && p != nil && p.HitFreezeFrames > 0 {
										freezeFrames = p.HitFreezeFrames
									}
									if existing, ok := ecs.Get(w, e, component.HitFreezeRequestComponent.Kind()); ok && existing != nil && existing.Frames > freezeFrames {
										freezeFrames = existing.Frames
									}
									_ = ecs.Add(w, e, component.HitFreezeRequestComponent.Kind(), &component.HitFreezeRequest{Frames: freezeFrames})

									// Add a transient HitEvent on the attacker so the player's
									// attack state can detect the successful hit and play
									// the local 'hit' SFX. This avoids directly manipulating
									// audio here and keeps the attack-state logic in one place.
									_ = ecs.Add(w, e, component.HitEventComponent.Kind(), &component.HitEvent{})
								}
							}
						}
					}
				})
			}
		},
	)
}
