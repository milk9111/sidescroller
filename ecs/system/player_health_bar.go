package system

import (
	"strconv"

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

	gearCount := currentPlayerGearCount(w)
	if barEntity, ok := ecs.First(w, component.PlayerHealthBarComponent.Kind()); ok {
		bar, ok := ecs.Get(w, barEntity, component.PlayerHealthBarComponent.Kind())
		if !ok || bar == nil {
			return
		}
		hud, ok := ecs.Get(w, barEntity, component.PlayerHUDUIComponent.Kind())
		if !ok || hud == nil {
			return
		}

		if bar.LastHealth != current {
			for index, heart := range hud.Hearts {
				if heart == nil {
					continue
				}
				if index < current {
					heart.Image = hud.HeartFullImage
				} else {
					heart.Image = hud.HeartEmptyImage
				}
			}
			bar.LastHealth = current
		}

		if bar.LastGearCount != gearCount {
			if hud.GearText != nil {
				hud.GearText.Label = strconv.Itoa(gearCount)
			}
			if hud.Root != nil {
				hud.Root.RequestRelayout()
			}
			bar.LastGearCount = gearCount
		}

		_ = ecs.Add(w, barEntity, component.PlayerHealthBarComponent.Kind(), bar)
	}
}
