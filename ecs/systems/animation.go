package systems

import (
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/ecs/render"
)

// AnimationSystem updates animator components.
type AnimationSystem struct {
	Library *render.AnimationLibrary
}

// NewAnimationSystem creates an AnimationSystem.
func NewAnimationSystem(lib *render.AnimationLibrary) *AnimationSystem {
	return &AnimationSystem{Library: lib}
}

// Update advances animations and binds events on first use.
func (s *AnimationSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	set := w.Animators()
	if set == nil {
		return
	}
	for _, id := range set.Entities() {
		v := set.Get(id)
		anim, ok := v.(*components.Animator)
		if !ok || anim == nil {
			continue
		}
		if anim.Anim == nil && anim.ClipKey != "" && s.Library != nil {
			if clip, ok := s.Library.Get(anim.ClipKey); ok {
				anim.Anim = clip.Anim
				anim.EventMap = clip.Events
				if anim.EventMap != nil && anim.Emitter != nil {
					component.BindAnimationEvents(anim.Anim, anim.EventMap, anim.Emitter)
				}
			}
		}
		if anim.Anim == nil {
			continue
		}
		anim.Anim.Update()
	}
}
