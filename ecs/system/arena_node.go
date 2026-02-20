package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type ArenaNodeSystem struct{}

func NewArenaNodeSystem() *ArenaNodeSystem { return &ArenaNodeSystem{} }

func (s *ArenaNodeSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach(w, component.ArenaNodeComponent.Kind(), func(e ecs.Entity, node *component.ArenaNode) {
		if node == nil {
			return
		}

		rt, ok := ecs.Get(w, e, component.ArenaNodeRuntimeComponent.Kind())
		if !ok || rt == nil {
			rt = &component.ArenaNodeRuntime{}
		}

		if !rt.Initialized {
			if hz, ok := ecs.Get(w, e, component.HazardComponent.Kind()); ok && hz != nil {
				rt.HasHazardTemplate = true
				rt.HazardTemplate = *hz
			}
			if tr, ok := ecs.Get(w, e, component.TransitionComponent.Kind()); ok && tr != nil {
				rt.HasTransitionTemplate = true
				rt.TransitionTemplate = *tr
			}
			rt.Initialized = true
		}

		if !node.Active {
			_ = ecs.Remove(w, e, component.HazardComponent.Kind())
			_ = ecs.Remove(w, e, component.TransitionComponent.Kind())
			_ = ecs.Add(w, e, component.ArenaNodeRuntimeComponent.Kind(), rt)
			return
		}

		if node.HazardEnabled {
			if rt.HasHazardTemplate && !ecs.Has(w, e, component.HazardComponent.Kind()) {
				t := rt.HazardTemplate
				_ = ecs.Add(w, e, component.HazardComponent.Kind(), &t)
			}
		} else {
			_ = ecs.Remove(w, e, component.HazardComponent.Kind())
		}

		if node.TransitionEnabled {
			if rt.HasTransitionTemplate && !ecs.Has(w, e, component.TransitionComponent.Kind()) {
				t := rt.TransitionTemplate
				_ = ecs.Add(w, e, component.TransitionComponent.Kind(), &t)
			}
		} else {
			_ = ecs.Remove(w, e, component.TransitionComponent.Kind())
		}

		_ = ecs.Add(w, e, component.ArenaNodeRuntimeComponent.Kind(), rt)
	})
}
