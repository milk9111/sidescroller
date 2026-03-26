package system

import (
	"math/rand"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type SpriteShakeSystem struct{}

func NewSpriteShakeSystem() *SpriteShakeSystem { return &SpriteShakeSystem{} }

func (s *SpriteShakeSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach(w, component.SpriteShakeComponent.Kind(), func(e ecs.Entity, shake *component.SpriteShake) {
		if shake == nil {
			return
		}

		if shake.Frames <= 0 || shake.Intensity <= 0 {
			markStaticTileBatchDirty(w, e)
			_ = ecs.Remove(w, e, component.SpriteShakeComponent.Kind())
			return
		}

		markStaticTileBatchDirty(w, e)
		shake.OffsetX = (rand.Float64()*2 - 1) * shake.Intensity
		shake.OffsetY = (rand.Float64()*2 - 1) * shake.Intensity
		shake.Frames--
		if shake.Frames <= 0 {
			markStaticTileBatchDirty(w, e)
			_ = ecs.Remove(w, e, component.SpriteShakeComponent.Kind())
		}
	})
}

func markStaticTileBatchDirty(w *ecs.World, e ecs.Entity) {
	if w == nil || !ecs.Has(w, e, component.StaticTileComponent.Kind()) {
		return
	}
	if boundsEntity, ok := ecs.First(w, component.LevelGridComponent.Kind()); ok {
		if state, ok := ecs.Get(w, boundsEntity, component.StaticTileBatchStateComponent.Kind()); ok && state != nil {
			state.Dirty = true
		}
	}
}
