package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
)

type TutorialSystem struct{}

func NewTutorialSystem() *TutorialSystem {
	return &TutorialSystem{}
}

func (s *TutorialSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ent, ok := ecs.First(w, component.TutorialStateComponent.Kind())
	if !ok {
		return
	}

	state, ok := ecs.Get(w, ent, component.TutorialStateComponent.Kind())
	if !ok || state == nil || !state.Active {
		return
	}

	if state.RemainingFrames < 0 {
		return
	}
	if state.RemainingFrames == 0 {
		_ = levelentity.HideTutorial(w)
		return
	}

	state.RemainingFrames--
	if state.RemainingFrames > 0 {
		return
	}

	_ = levelentity.HideTutorial(w)
}
