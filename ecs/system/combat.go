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

			// Compute hitbox world AABB
			hx := transform.X + hb.OffsetX
			hy := transform.Y + hb.OffsetY
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
						// Apply damage if target has health
						if h, ok := ecs.Get(w, t, component.HealthComponent); ok {
							h.Current -= hb.Damage
							if h.Current < 0 {
								h.Current = 0
								fmt.Println("Entity", t, "defeated!")
							}
							ecs.Add(w, t, component.HealthComponent, h)
						}
					}
				}
			}
		}
	}
}
