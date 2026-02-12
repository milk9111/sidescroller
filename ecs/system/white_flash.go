package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type WhiteFlashSystem struct{}

func NewWhiteFlashSystem() *WhiteFlashSystem { return &WhiteFlashSystem{} }

func (s *WhiteFlashSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach(w, component.WhiteFlashComponent.Kind(), func(e ecs.Entity, a *component.WhiteFlash) {
		wf, ok := ecs.Get(w, e, component.WhiteFlashComponent.Kind())
		if !ok {
			return
		}

		if wf.Interval <= 0 {
			wf.Interval = 1
		}

		wf.Timer++
		if wf.Timer >= wf.Interval {
			wf.Timer = 0
			wf.On = !wf.On
			wf.Frames -= wf.Interval
		}

		if wf.Frames <= 0 {
			_ = ecs.Remove(w, e, component.WhiteFlashComponent.Kind())
		} else {
			//_ = ecs.Add(w, e, component.WhiteFlashComponent.Kind(), wf)
		}
	})
}
