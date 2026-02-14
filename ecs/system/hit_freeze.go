package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type HitFreezeSystem struct {
	onFreeze func(frames int)
}

func NewHitFreezeSystem(onFreeze func(frames int)) *HitFreezeSystem {
	return &HitFreezeSystem{onFreeze: onFreeze}
}

func (s *HitFreezeSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	maxFrames := 0
	ecs.ForEach(w, component.HitFreezeRequestComponent.Kind(), func(e ecs.Entity, req *component.HitFreezeRequest) {
		if req != nil && req.Frames > maxFrames {
			maxFrames = req.Frames
		}
		_ = ecs.Remove(w, e, component.HitFreezeRequestComponent.Kind())
	})

	if maxFrames > 0 && s.onFreeze != nil {
		s.onFreeze(maxFrames)
	}
}
