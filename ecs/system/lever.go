package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type LeverSystem struct{}

func NewLeverSystem() *LeverSystem { return &LeverSystem{} }

func (s *LeverSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach2(w, component.LeverComponent.Kind(), component.AnimationComponent.Kind(), func(e ecs.Entity, lever *component.Lever, anim *component.Animation) {
		if lever == nil || anim == nil {
			return
		}

		req, hasHitRequest := ecs.Get(w, e, component.LeverHitRequestComponent.Kind())
		if hasHitRequest && req != nil {
			_ = ecs.Remove(w, e, component.LeverHitRequestComponent.Kind())
		}

		switch lever.State {
		case "", component.LeverStateOpen:
			lever.State = component.LeverStateOpen
			setLeverAnimation(lever, anim, component.LeverStateOpen)
			if hasHitRequest && req != nil {
				lever.State = component.LeverStateClosing
				setLeverAnimation(lever, anim, component.LeverStateClosing)
				recordLevelEntityState(w, e, component.PersistedLevelEntityStateUsed)
			}
		case component.LeverStateClosing:
			setLeverAnimation(lever, anim, component.LeverStateClosing)
			if leverAnimationFinished(lever, anim) {
				lever.State = component.LeverStateClosed
				setLeverAnimation(lever, anim, component.LeverStateClosed)
				emitLeverClosedSignal(w, e)
			}
		case component.LeverStateClosed:
			setLeverAnimation(lever, anim, component.LeverStateClosed)
		default:
			lever.State = component.LeverStateOpen
			setLeverAnimation(lever, anim, component.LeverStateOpen)
		}
	})
}

func applyLeverPersistedState(w *ecs.World, e ecs.Entity) {
	if w == nil {
		return
	}

	lever, ok := ecs.Get(w, e, component.LeverComponent.Kind())
	if !ok || lever == nil {
		return
	}

	lever.State = component.LeverStateClosed
	if anim, ok := ecs.Get(w, e, component.AnimationComponent.Kind()); ok && anim != nil {
		setLeverAnimation(lever, anim, component.LeverStateClosed)
	}
}

func setLeverAnimation(lever *component.Lever, anim *component.Animation, state component.LeverState) {
	if lever == nil || anim == nil {
		return
	}

	name := leverAnimationName(lever, state)
	if name == "" {
		return
	}

	def, ok := anim.Defs[name]
	if !ok {
		return
	}

	if anim.Current == name {
		if state == component.LeverStateClosing {
			return
		}
		if anim.Playing {
			return
		}
	}

	anim.Current = name
	anim.FrameTimer = 0
	anim.FrameProgress = 0
	anim.Frame = 0
	anim.Playing = true
	if !def.Loop && def.FrameCount <= 1 {
		anim.Playing = false
	}
	if def.FrameCount <= 1 {
		anim.Frame = 0
	}
}

func leverAnimationFinished(lever *component.Lever, anim *component.Animation) bool {
	if lever == nil || anim == nil {
		return false
	}

	name := leverAnimationName(lever, component.LeverStateClosing)
	if name == "" || anim.Current != name {
		return false
	}

	def, ok := anim.Defs[name]
	if !ok || def.FrameCount <= 0 {
		return true
	}

	return !anim.Playing && anim.Frame == def.FrameCount-1
}

func leverAnimationName(lever *component.Lever, state component.LeverState) string {
	if lever == nil {
		return ""
	}

	switch state {
	case component.LeverStateOpen:
		return lever.OpenAnimation
	case component.LeverStateClosing:
		return lever.ClosingAnimation
	case component.LeverStateClosed:
		return lever.ClosedAnimation
	default:
		return ""
	}
}

func emitLeverClosedSignal(w *ecs.World, e ecs.Entity) {
	if w == nil || !ecs.IsAlive(w, e) {
		return
	}

	if transform, ok := ecs.Get(w, e, component.TransformComponent.Kind()); ok && transform != nil {
		BroadcastSignalWithPosition(w, e, "on_lever_closed", transform.X, transform.Y, true)
		return
	}

	BroadcastSignalWithPosition(w, e, "on_lever_closed", 0, 0, false)
}
