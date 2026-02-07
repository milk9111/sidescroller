package system

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AnimationSystem struct {
	Animation component.ComponentHandle[component.Animation]
	Sprite    component.ComponentHandle[component.Sprite]
}

func NewAnimationSystem(animation component.ComponentHandle[component.Animation], sprite component.ComponentHandle[component.Sprite]) *AnimationSystem {
	return &AnimationSystem{Animation: animation, Sprite: sprite}
}

func (a *AnimationSystem) Update(w *ecs.World) {
	for _, e := range w.Query(a.Animation.Kind(), a.Sprite.Kind()) {
		anim, ok := ecs.Get(w, e, a.Animation)
		if !ok || anim.Sheet == nil || !anim.Playing {
			continue
		}
		def, ok := anim.Defs[anim.Current]
		if !ok || def.FrameCount <= 0 {
			continue
		}
		// Advance frame every N ticks based on FPS and 60 TPS
		ticksPerFrame := int(60.0 / def.FPS)
		if ticksPerFrame < 1 {
			ticksPerFrame = 1
		}
		anim.FrameTimer++
		if int(anim.FrameTimer) >= ticksPerFrame {
			anim.FrameTimer = 0
			anim.Frame++
			if anim.Frame >= def.FrameCount {
				if def.Loop {
					anim.Frame = 0
				} else {
					anim.Frame = def.FrameCount - 1
					anim.Playing = false
				}
			}
		}
		// Calculate subimage rect
		x := def.ColStart*def.FrameW + anim.Frame*def.FrameW
		y := def.Row * def.FrameH
		rect := image.Rect(x, y, x+def.FrameW, y+def.FrameH)
		sprite, ok := ecs.Get(w, e, a.Sprite)
		if ok {
			sprite.Image = anim.Sheet.SubImage(rect).(*ebiten.Image)
			ecs.Add(w, e, a.Sprite, sprite)
		}
		// Write back anim state
		ecs.Add(w, e, a.Animation, anim)
	}
}
