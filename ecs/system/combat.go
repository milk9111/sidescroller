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
	for _, e := range w.Query(component.HitboxComponent.Kind(), component.TransformComponent.Kind(), component.AnimationComponent.Kind()) {
		hitboxes, _ := ecs.Get(w, e, component.HitboxComponent)
		transform, _ := ecs.Get(w, e, component.TransformComponent)
		anim, _ := ecs.Get(w, e, component.AnimationComponent)

		for _, hb := range hitboxes {
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
			if s, ok := ecs.Get(w, e, component.SpriteComponent); ok && s.FacingLeft {
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

			// Check against all hurtboxes
			for _, t := range w.Query(component.HurtboxComponent.Kind(), component.TransformComponent.Kind()) {
				if t == e {
					continue
				}
				hurtboxes, _ := ecs.Get(w, t, component.HurtboxComponent)
				tTransform, _ := ecs.Get(w, t, component.TransformComponent)

				for _, hurt := range hurtboxes {
					tx := tTransform.X + hurt.OffsetX
					ty := tTransform.Y + hurt.OffsetY
					tw := hurt.Width
					th := hurt.Height

					if intersects(hx, hy, hw, hh, tx, ty, tw, th) {
						// Skip if target is temporarily invulnerable
						if ecs.Has(w, t, component.InvulnerableComponent) {
							continue
						}
						// Apply damage if target has health
						if h, ok := ecs.Get(w, t, component.HealthComponent); ok {
							h.Current -= hb.Damage
							if h.Current < 0 {
								h.Current = 0
								fmt.Println("Entity", t, "defeated!")
							}
							ecs.Add(w, t, component.HealthComponent, h)
							// If this target is a player, send a state interrupt requesting the
							// hit state so the controller can transition the player.
							if ecs.Has(w, t, component.PlayerTagComponent) {
								err := ecs.Add(w, t, component.PlayerStateInterruptComponent, component.PlayerStateInterrupt{State: "hit"})
								if err != nil {
									panic("combat: add player state interrupt: " + err.Error())
								}
							}
							// If this target is an AI (enemy), request the AI FSM handle a 'hit' event
							if ecs.Has(w, t, component.AITagComponent) {
								err := ecs.Add(w, t, component.AIStateInterruptComponent, component.AIStateInterrupt{Event: "hit"})
								if err != nil {
									panic("combat: add ai state interrupt: " + err.Error())
								}
							}
						}
					}
				}
			}
		}
	}
}
