package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	healthBarPaddingX = 12.0
	healthBarPaddingY = 12.0
	heartSpacing      = 4.0
	healthBarLayer    = 1000
)

func NewPlayerHealthBar(w *ecs.World) (ecs.Entity, error) {
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return 0, nil
	}

	health, ok := ecs.Get(w, player, component.HealthComponent.Kind())
	if !ok || health == nil || health.Initial <= 0 {
		return 0, nil
	}

	heartImage, err := assets.LoadImage("life_heart.png")
	if err != nil {
		return 0, fmt.Errorf("player health bar: load heart sprite: %w", err)
	}

	heartW := float64(heartImage.Bounds().Dx())
	barEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, barEntity, component.PlayerHealthBarComponent.Kind(), &component.PlayerHealthBar{MaxHearts: health.Initial}); err != nil {
		return 0, fmt.Errorf("player health bar: add bar component: %w", err)
	}
	if err := ecs.Add(w, barEntity, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{}); err != nil {
		return 0, fmt.Errorf("player health bar: add screen-space: %w", err)
	}
	if err := ecs.Add(w, barEntity, component.TransformComponent.Kind(), &component.Transform{X: healthBarPaddingX, Y: healthBarPaddingY, ScaleX: 1, ScaleY: 1}); err != nil {
		return 0, fmt.Errorf("player health bar: add transform: %w", err)
	}

	for i := 0; i < health.Initial; i++ {
		heartEntity := ecs.CreateEntity(w)
		x := healthBarPaddingX + float64(i)*(heartW+heartSpacing)
		if err := ecs.Add(w, heartEntity, component.PlayerHealthHeartComponent.Kind(), &component.PlayerHealthHeart{Slot: i}); err != nil {
			return 0, fmt.Errorf("player health bar: add heart component: %w", err)
		}
		if err := ecs.Add(w, heartEntity, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{}); err != nil {
			return 0, fmt.Errorf("player health bar: add heart screen-space: %w", err)
		}
		if err := ecs.Add(w, heartEntity, component.TransformComponent.Kind(), &component.Transform{X: x, Y: healthBarPaddingY, ScaleX: 1, ScaleY: 1}); err != nil {
			return 0, fmt.Errorf("player health bar: add heart transform: %w", err)
		}
		if err := ecs.Add(w, heartEntity, component.SpriteComponent.Kind(), &component.Sprite{Image: heartImage}); err != nil {
			return 0, fmt.Errorf("player health bar: add heart sprite: %w", err)
		}
		if err := ecs.Add(w, heartEntity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: healthBarLayer}); err != nil {
			return 0, fmt.Errorf("player health bar: add heart render layer: %w", err)
		}
	}

	return barEntity, nil
}
