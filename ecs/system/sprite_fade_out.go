package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type SpriteFadeOutSystem struct{}

func NewSpriteFadeOutSystem() *SpriteFadeOutSystem { return &SpriteFadeOutSystem{} }

func (s *SpriteFadeOutSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach(w, component.SpriteFadeOutComponent.Kind(), func(e ecs.Entity, fade *component.SpriteFadeOut) {
		if fade == nil {
			return
		}

		if fade.TotalFrames <= 0 {
			fade.TotalFrames = fade.Frames
		}
		if fade.TotalFrames <= 0 {
			markStaticTileBatchDirty(w, e)
			_ = ecs.Remove(w, e, component.SpriteFadeOutComponent.Kind())
			return
		}

		markStaticTileBatchDirty(w, e)
		fade.Frames--
		if fade.Frames < 0 {
			markStaticTileBatchDirty(w, e)
			_ = ecs.Remove(w, e, component.SpriteFadeOutComponent.Kind())
			return
		}

		fade.Alpha = clampColor01(float64(fade.Frames) / float64(fade.TotalFrames))
	})
}