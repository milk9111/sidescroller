package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const defaultItemRange = 196.0
const itemPopupVerticalGap = 6.0

type ItemPopupSystem struct{}

func NewItemPopupSystem() *ItemPopupSystem {
	return &ItemPopupSystem{}
}

func (s *ItemPopupSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	inputEntity, ok := ecs.First(w, component.DialogueInputComponent.Kind())
	if !ok {
		return
	}
	input, ok := ecs.Get(w, inputEntity, component.DialogueInputComponent.Kind())
	if !ok || input == nil {
		return
	}

	popupEntity, ok := ecs.First(w, component.ItemPopupComponent.Kind())
	if !ok {
		return
	}
	popup, ok := ecs.Get(w, popupEntity, component.ItemPopupComponent.Kind())
	if !ok || popup == nil {
		return
	}

	popupSprite, ok := ecs.Get(w, popupEntity, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		return
	}

	playerX, playerY, ok := playerWorldPosition(w)
	if !ok {
		popup.TargetItemEntity = 0
		popupSprite.Disabled = true
		return
	}

	bestDistanceSq := math.MaxFloat64
	bestEntity := ecs.Entity(0)
	bestAnchorX := 0.0
	bestAnchorY := 0.0
	found := false

	ecs.ForEach3(w, component.ItemComponent.Kind(), component.TransformComponent.Kind(), component.SpriteComponent.Kind(), func(e ecs.Entity, item *component.Item, transform *component.Transform, sprite *component.Sprite) {
		if item == nil || transform == nil || sprite == nil || sprite.Image == nil || sprite.Disabled {
			return
		}

		rangeLimit := item.Range
		if rangeLimit <= 0 {
			rangeLimit = defaultItemRange
		}

		itemX, itemY, ok := entityWorldPosition(w, e)
		if !ok {
			return
		}

		dx := itemX - playerX
		dy := itemY - playerY
		distanceSq := dx*dx + dy*dy
		if distanceSq > rangeLimit*rangeLimit {
			return
		}

		anchorX, anchorY, ok := spriteTopCenterWorld(transform, sprite)
		if !ok {
			return
		}

		if found && distanceSq >= bestDistanceSq {
			return
		}

		found = true
		bestDistanceSq = distanceSq
		bestEntity = e
		bestAnchorX = anchorX
		bestAnchorY = anchorY
	})

	if !found {
		popup.TargetItemEntity = 0
		popupSprite.Disabled = true
		return
	}

	popupTransform, ok := ecs.Get(w, popupEntity, component.TransformComponent.Kind())
	if !ok || popupTransform == nil {
		popupTransform = &component.Transform{ScaleX: 1, ScaleY: 1}
		if err := ecs.Add(w, popupEntity, component.TransformComponent.Kind(), popupTransform); err != nil {
			return
		}
	}

	if !popup.HasRenderedImage || popup.RenderedGamepad != input.UsingGamepad {
		popupSprite.Image = composePopupImage(popup.Base, popup.KeyboardCue)
		if input.UsingGamepad {
			popupSprite.Image = composePopupImage(popup.Base, popup.GamepadCue)
		}
		popup.HasRenderedImage = true
		popup.RenderedGamepad = input.UsingGamepad
	}

	if popupSprite.Image != nil && popupSprite.OriginX == 0 && popupSprite.OriginY == 0 {
		bounds := popupSprite.Image.Bounds()
		if bounds.Dx() > 0 && bounds.Dy() > 0 {
			popupSprite.OriginX = float64(bounds.Dx()) / 2
			popupSprite.OriginY = float64(bounds.Dy())
		}
	}

	popup.TargetItemEntity = uint64(bestEntity)
	popupSprite.Disabled = false
	popupTransform.X = bestAnchorX
	popupTransform.Y = bestAnchorY - itemPopupVerticalGap
	popupTransform.Rotation = 0
	if popupTransform.ScaleX == 0 {
		popupTransform.ScaleX = 1
	}
	if popupTransform.ScaleY == 0 {
		popupTransform.ScaleY = 1
	}
}
