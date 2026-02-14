package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type PlayerHealthBarSystem struct{}

func NewPlayerHealthBarSystem() *PlayerHealthBarSystem { return &PlayerHealthBarSystem{} }

func (s *PlayerHealthBarSystem) Update(w *ecs.World) {
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	health, ok := ecs.Get(w, player, component.HealthComponent.Kind())
	if !ok || health == nil || health.Initial <= 0 {
		return
	}

	current := health.Current
	if current < 0 {
		current = 0
	}
	if current > health.Initial {
		current = health.Initial
	}

	ecs.ForEach(w, component.PlayerHealthHeartComponent.Kind(), func(e ecs.Entity, heart *component.PlayerHealthHeart) {
		if heart == nil {
			return
		}

		shouldBlackout := heart.Slot >= current
		hasBlackout := ecs.Has(w, e, component.SpriteBlackoutComponent.Kind())

		switch {
		case shouldBlackout && !hasBlackout:
			_ = ecs.Add(w, e, component.SpriteBlackoutComponent.Kind(), &component.SpriteBlackout{})
		case !shouldBlackout && hasBlackout:
			ecs.Remove(w, e, component.SpriteBlackoutComponent.Kind())
		}
	})
}
