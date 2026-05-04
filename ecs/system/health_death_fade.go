package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const defaultHealthDeathFadeFrames = 10
const defaultHealthDeathPostAnimationFrames = 60

type HealthDeathFadeSystem struct {
	FadeFrames          int
	PostAnimationFrames int
}

func NewHealthDeathFadeSystem() *HealthDeathFadeSystem {
	return &HealthDeathFadeSystem{
		FadeFrames:          defaultHealthDeathFadeFrames,
		PostAnimationFrames: defaultHealthDeathPostAnimationFrames,
	}
}

func (s *HealthDeathFadeSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	fadeFrames := s.FadeFrames
	if fadeFrames <= 0 {
		fadeFrames = defaultHealthDeathFadeFrames
	}
	postAnimationFrames := s.PostAnimationFrames
	if postAnimationFrames <= 0 {
		postAnimationFrames = defaultHealthDeathPostAnimationFrames
	}

	ecs.ForEach(w, component.HealthComponent.Kind(), func(e ecs.Entity, health *component.Health) {
		if health == nil || health.Current > 0 {
			return
		}
		if ecs.Has(w, e, component.PlayerTagComponent.Kind()) {
			return
		}

		state, ok := ecs.Get(w, e, component.HealthDeathFadeComponent.Kind())
		if !ok || state == nil {
			state = &component.HealthDeathFade{FadeFrames: fadeFrames}
			_ = ecs.Add(w, e, component.HealthDeathFadeComponent.Kind(), state)
		}

		if !state.FadeStarted {
			waitingForAnimation, hasDeathAnimation := shouldWaitForDeathAnimation(w, e)
			if waitingForAnimation {
				return
			}
			if hasDeathAnimation && !state.PostAnimationArmed {
				state.PostAnimationFrames = postAnimationFrames
				state.PostAnimationArmed = true
				return
			}
			if state.PostAnimationFrames > 0 {
				state.PostAnimationFrames--
				return
			}
			if !startHealthDeathFade(w, e, state) {
				ecs.DestroyEntity(w, e)
				return
			}
		}

		if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
			return
		}

		ecs.DestroyEntity(w, e)
	})
}

func shouldWaitForDeathAnimation(w *ecs.World, e ecs.Entity) (bool, bool) {
	anim, ok := ecs.Get(w, e, component.AnimationComponent.Kind())
	if !ok || anim == nil {
		return false, false
	}

	def, ok := anim.Defs["death"]
	if !ok || def.FrameCount <= 0 || def.Loop {
		return false, false
	}

	if anim.Current != "death" {
		anim.Current = "death"
		anim.Frame = 0
		anim.FrameTimer = 0
		anim.FrameProgress = 0
		anim.Playing = true
		if def.FrameCount <= 1 {
			anim.Frame = 0
			anim.Playing = false
		}
	}

	return anim.Playing || anim.Frame != def.FrameCount-1, true
}

func startHealthDeathFade(w *ecs.World, e ecs.Entity, state *component.HealthDeathFade) bool {
	if state == nil {
		return false
	}

	state.FadeStarted = true
	if fade, ok := ecs.Get(w, e, component.SpriteFadeOutComponent.Kind()); ok && fade != nil {
		return true
	}
	if !ecs.Has(w, e, component.SpriteComponent.Kind()) {
		return false
	}

	frames := state.FadeFrames
	if frames <= 0 {
		frames = defaultHealthDeathFadeFrames
	}
	_ = ecs.Add(w, e, component.SpriteFadeOutComponent.Kind(), &component.SpriteFadeOut{
		Frames:      frames,
		TotalFrames: frames,
		Alpha:       1,
	})
	return true
}
