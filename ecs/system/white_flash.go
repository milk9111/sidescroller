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

	for _, e := range w.Query(component.WhiteFlashComponent.Kind()) {
		wf, ok := ecs.Get(w, e, component.WhiteFlashComponent)
		if !ok {
			continue
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
			_ = ecs.Remove(w, e, component.WhiteFlashComponent)
		} else {
			_ = ecs.Add(w, e, component.WhiteFlashComponent, wf)
		}
	}
}
