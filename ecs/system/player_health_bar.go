package system

import (
	"strconv"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const playerHUDHealMaxUses = 2

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
	healUses, canHeal := currentPlayerHealState(w, player)
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

		if bar.LastHealUses != healUses || bar.LastCanHeal != canHeal {
			for index, flask := range hud.Flasks {
				if flask == nil {
					continue
				}
				if canHeal {
					flask.GetWidget().Visibility = widget.Visibility_Show
				} else {
					flask.GetWidget().Visibility = widget.Visibility_Hide
				}
				if !canHeal || index < healUses {
					flask.Image = hud.FlaskEmptyImage
				} else {
					flask.Image = hud.FlaskFullImage
				}
			}
			if hud.Root != nil {
				hud.Root.RequestRelayout()
			}
			bar.LastHealUses = healUses
			bar.LastCanHeal = canHeal
		}

		_ = ecs.Add(w, barEntity, component.PlayerHealthBarComponent.Kind(), bar)
	}
}

func currentPlayerHealState(w *ecs.World, player ecs.Entity) (healUses int, canHeal bool) {
	if abilitiesEntity, ok := ecs.First(w, component.AbilitiesComponent.Kind()); ok {
		if abilities, ok := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind()); ok && abilities != nil {
			canHeal = abilities.Heal
		}
	}
	if stateMachine, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind()); ok && stateMachine != nil {
		healUses = stateMachine.HealUses
	}
	if healUses < 0 {
		healUses = 0
	}
	if healUses > playerHUDHealMaxUses {
		healUses = playerHUDHealMaxUses
	}
	return healUses, canHeal
}
