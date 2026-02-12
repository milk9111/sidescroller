package system

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AnimationSystem struct{}

func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

func (a *AnimationSystem) Update(w *ecs.World) {
	ecs.ForEach2(w, component.AnimationComponent.Kind(), component.SpriteComponent.Kind(), func(e ecs.Entity, anim *component.Animation, sprite *component.Sprite) {
		anim, ok := ecs.Get(w, e, component.AnimationComponent.Kind())
		if !ok || anim.Sheet == nil || !anim.Playing {
			return
		}

		def, ok := anim.Defs[anim.Current]
		if !ok || def.FrameCount <= 0 {
			return
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
		sprite.Image = anim.Sheet.SubImage(rect).(*ebiten.Image)

		// ecs.Add(w, e, component.SpriteComponent.Kind(), sprite)
		// ecs.Add(w, e, component.AnimationComponent.Kind(), anim)
	})
}
