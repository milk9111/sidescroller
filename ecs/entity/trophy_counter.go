package entity

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	trophyCounterLayer = 1000
)

func NewTrophyCounter(w *ecs.World) (ecs.Entity, error) {
	trophyImage, err := assets.LoadImage("trophy.png")
	if err != nil {
		return 0, fmt.Errorf("trophy counter: load trophy sprite: %w", err)
	}

	counterEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, counterEntity, component.TrophyCounterComponent.Kind(), &component.TrophyCounter{}); err != nil {
		return 0, fmt.Errorf("trophy counter: add counter component: %w", err)
	}

	iconEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, iconEntity, component.TrophyCounterIconComponent.Kind(), &component.TrophyCounterIcon{}); err != nil {
		return 0, fmt.Errorf("trophy counter: add icon component: %w", err)
	}
	if err := ecs.Add(w, iconEntity, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{}); err != nil {
		return 0, fmt.Errorf("trophy counter: add icon screen-space: %w", err)
	}
	if err := ecs.Add(w, iconEntity, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		return 0, fmt.Errorf("trophy counter: add icon transform: %w", err)
	}
	if err := ecs.Add(w, iconEntity, component.SpriteComponent.Kind(), &component.Sprite{Image: trophyImage}); err != nil {
		return 0, fmt.Errorf("trophy counter: add icon sprite: %w", err)
	}
	if err := ecs.Add(w, iconEntity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: trophyCounterLayer}); err != nil {
		return 0, fmt.Errorf("trophy counter: add icon layer: %w", err)
	}

	textEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, textEntity, component.TrophyCounterTextComponent.Kind(), &component.TrophyCounterText{}); err != nil {
		return 0, fmt.Errorf("trophy counter: add text component: %w", err)
	}
	if err := ecs.Add(w, textEntity, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{}); err != nil {
		return 0, fmt.Errorf("trophy counter: add text screen-space: %w", err)
	}
	if err := ecs.Add(w, textEntity, component.TransformComponent.Kind(), &component.Transform{ScaleX: 1, ScaleY: 1}); err != nil {
		return 0, fmt.Errorf("trophy counter: add text transform: %w", err)
	}
	if err := ecs.Add(w, textEntity, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(1, 1)}); err != nil {
		return 0, fmt.Errorf("trophy counter: add text sprite: %w", err)
	}
	if err := ecs.Add(w, textEntity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: trophyCounterLayer}); err != nil {
		return 0, fmt.Errorf("trophy counter: add text layer: %w", err)
	}

	return counterEntity, nil
}
