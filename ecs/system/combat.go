package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type CombatSystem struct{}

func NewCombatSystem() *CombatSystem { return &CombatSystem{} }

func intersects(ax, ay, aw, ah, bx, by, bw, bh float64) (float64, float64, bool) {
	if ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by {
		// compute intersection rect
		ix := math.Max(ax, bx)
		iy := math.Max(ay, by)
		iw := math.Min(ax+aw, bx+bw) - ix
		ih := math.Min(ay+ah, by+bh) - iy
		// return center of intersection area
		return ix + iw/2, iy + ih/2, true
	}
	return 0, 0, false
}

func frameActive(frames []int, frame int) bool {
	for _, f := range frames {
		if f == frame {
			return true
		}
	}
	return false
}

func blockedBeforeHurtbox(w *ecs.World, attacker ecs.Entity, x0, y0, x1, y1, hurtboxX, hurtboxY, hurtboxW, hurtboxH float64) bool {
	hitX, hitY, hit, _ := firstStaticHit(w, attacker, x0, y0, x1, y1)
	if !hit {
		return false
	}

	dx := x1 - x0
	dy := y1 - y0
	hurtboxHit, tHurtbox := segmentAABBHit(x0, y0, dx, dy, hurtboxX, hurtboxY, hurtboxX+hurtboxW, hurtboxY+hurtboxH)
	if !hurtboxHit {
		return false
	}

	tStatic := hitParam(x0, y0, x1, y1, hitX, hitY)
	const eps = 1e-6
	return tStatic > eps && tStatic < tHurtbox-eps
}

func (s *CombatSystem) Update(w *ecs.World) {
	// For each entity that has hitboxes, check configured frames and test against all hurtboxes
	ecs.ForEach3(
		w,
		component.HitboxComponent.Kind(),
		component.TransformComponent.Kind(),
		component.AnimationComponent.Kind(),
		func(e ecs.Entity, hitboxes *[]component.Hitbox, transform *component.Transform, anim *component.Animation) {
			// iterate by index so we can clear/mark per-hit state on the stored hitbox
			for i := range *hitboxes {
				hb := &(*hitboxes)[i]

				// Determine whether this hitbox is currently active for the entity's animation/frame.
				active := true
				if hb.Anim != "" && hb.Anim != anim.Current {
					active = false
				}
				if len(hb.Frames) > 0 && !frameActive(hb.Frames, anim.Frame) {
					active = false
				}

				// If hitbox is not active, clear the runtime hit tracking so it can hit again
				if !active {
					if hb.HitTargets != nil && len(hb.HitTargets) > 0 {
						hb.HitTargets = nil
					}
					continue
				}

				// Compute hitbox world AABB using centered local offsets.
				hx := aabbTopLeftX(w, e, transform.X, hb.OffsetX, hb.Width, false)
				hy := aabbTopLeftY(transform.Y, hb.OffsetY, hb.Height, false)
				hw := hb.Width
				hh := hb.Height

				ecs.ForEach3(w, component.HurtboxComponent.Kind(), component.TransformComponent.Kind(), component.HealthComponent.Kind(), func(et ecs.Entity, hurtboxes *[]component.Hurtbox, tTransform *component.Transform, health *component.Health) {
					if et == e {
						return
					}
					// Don't allow AI (enemies) to damage other AI — skip friendly fire between enemies.
					if ecs.Has(w, e, component.AITagComponent.Kind()) && ecs.Has(w, et, component.AITagComponent.Kind()) {
						return
					}
					for _, hurt := range *hurtboxes {
						tx := aabbTopLeftX(w, et, tTransform.X, hurt.OffsetX, hurt.Width, false)
						ty := aabbTopLeftY(tTransform.Y, hurt.OffsetY, hurt.Height, false)
						tw := hurt.Width
						th := hurt.Height

						if intersectionX, intersectionY, hit := intersects(hx, hy, hw, hh, tx, ty, tw, th); hit {
							blocked := blockedBeforeHurtbox(w, e, transform.X, transform.Y, intersectionX, intersectionY, tx, ty, tw, th)

							// Skip if target is temporarily invulnerable or if blocked by an earlier static obstacle
							if ecs.Has(w, et, component.InvulnerableComponent.Kind()) || blocked {
								continue
							}

							// Skip if target is already in the death state (no further damage)
							if sm, ok := ecs.Get(w, et, component.PlayerStateMachineComponent.Kind()); ok && sm.State != nil && sm.State.Name() == "death" {
								continue
							}

							// Prevent this hitbox from damaging the same entity multiple times
							if hb.HitTargets == nil {
								hb.HitTargets = make(map[uint64]bool)
							}
							if hb.HitTargets[uint64(et)] {
								continue
							}

							previousHealth := health.Current
							health.Current -= hb.Damage
							if health.Current < 0 {
								health.Current = 0
							}

							sourceX := hx + hw/2
							sourceY := hy + hh/2

							// mark entity as already hit by this hitbox during its current activation
							hb.HitTargets[uint64(et)] = true

							// Emit an `on_hit` signal so scripts/systems can respond to the hit
							EmitEntitySignal(w, et, e, "on_hit")
							if previousHealth > 0 && health.Current <= 0 {
								if ecs.Has(w, et, component.AITagComponent.Kind()) {
									recordLevelEntityState(w, et, component.PersistedLevelEntityStateDefeated)
								}
								EmitEntitySignal(w, et, e, "on_death")
							}
							if ecs.Has(w, et, component.PlayerTagComponent.Kind()) {
								req := &component.DamageKnockback{SourceX: sourceX, SourceY: sourceY, SourceEntity: uint64(e)}
								_ = ecs.Add(w, et, component.DamageKnockbackRequestComponent.Kind(), req)
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

							if ecs.Has(w, et, component.AITagComponent.Kind()) {
								strongKnockback := ecs.Has(w, e, component.PlayerTagComponent.Kind())
								req := &component.DamageKnockback{SourceX: sourceX, SourceY: sourceY, Strong: strongKnockback, SourceEntity: uint64(e)}
								_ = ecs.Add(w, et, component.DamageKnockbackRequestComponent.Kind(), req)
								err := ecs.Add(w, et, component.AIStateInterruptComponent.Kind(), &component.AIStateInterrupt{Event: "hit"})
								if err != nil {
									panic("combat: add ai state interrupt: " + err.Error())
								}
							}

							// If the player dealt damage to an enemy, request a short global hit-freeze.
							// if ecs.Has(w, e, component.PlayerTagComponent.Kind()) && ecs.Has(w, et, component.AITagComponent.Kind()) {
							// 	// Determine freeze frames from the attacker's player config if available.
							// 	freezeFrames := 5
							// 	if p, ok := ecs.Get(w, e, component.PlayerComponent.Kind()); ok && p != nil && p.HitFreezeFrames > 0 {
							// 		freezeFrames = p.HitFreezeFrames
							// 	}
							// 	if existing, ok := ecs.Get(w, e, component.HitFreezeRequestComponent.Kind()); ok && existing != nil && existing.Frames > freezeFrames {
							// 		freezeFrames = existing.Frames
							// 	}
							// 	_ = ecs.Add(w, e, component.HitFreezeRequestComponent.Kind(), &component.HitFreezeRequest{Frames: freezeFrames})

							// 	// Add a transient HitEvent on the attacker so the player's
							// 	// attack state can detect the successful hit and play
							// 	// the local 'hit' SFX. This avoids directly manipulating
							// 	// audio here and keeps the attack-state logic in one place.
							// 	_ = ecs.Add(w, e, component.HitEventComponent.Kind(), &component.HitEvent{})
							// }
						}
					}
				})
			}
		},
	)
}
