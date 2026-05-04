package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func gameplayTimeScale(w *ecs.World) float64 {
	if w == nil {
		return 1
	}

	ent, ok := ecs.First(w, component.GameplayTimeComponent.Kind())
	if !ok {
		return 1
	}

	time, ok := ecs.Get(w, ent, component.GameplayTimeComponent.Kind())
	if !ok || time == nil || time.Scale <= 0 {
		return 1
	}

	return time.Scale
}
