package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const transitionPopupVerticalGap = 6.0

type TransitionPopupSystem struct{}

func NewTransitionPopupSystem() *TransitionPopupSystem {
	return &TransitionPopupSystem{}
}

func (s *TransitionPopupSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	popupEntity, ok := ecs.First(w, component.TransitionPopupComponent.Kind())
	if !ok {
		return
	}

	popup, ok := ecs.Get(w, popupEntity, component.TransitionPopupComponent.Kind())
	if !ok || popup == nil {
		return
	}

	popupSprite, ok := ecs.Get(w, popupEntity, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		return
	}

	if _, active := ecs.First(w, component.TransitionRuntimeComponent.Kind()); active {
		hideTransitionPopup(popup, popupSprite)
		return
	}

	playerBounds, ok := playerBoundsForInsideTransition(w)
	if !ok {
		hideTransitionPopup(popup, popupSprite)
		return
	}

	transitionEntity, _, bounds, found := activeInsideTransition(w, playerBounds)
	if !found {
		hideTransitionPopup(popup, popupSprite)
		return
	}

	popupTransform, ok := ecs.Get(w, popupEntity, component.TransformComponent.Kind())
	if !ok || popupTransform == nil {
		popupTransform = &component.Transform{ScaleX: 1, ScaleY: 1}
		if err := ecs.Add(w, popupEntity, component.TransformComponent.Kind(), popupTransform); err != nil {
			return
		}
	}

	usingGamepad := false
	if inputEntity, ok := ecs.First(w, component.TransitionInputComponent.Kind()); ok {
		if input, ok := ecs.Get(w, inputEntity, component.TransitionInputComponent.Kind()); ok && input != nil {
			usingGamepad = input.UsingGamepad
		}
	}

	if !popup.HasRenderedImage || popup.RenderedGamepad != usingGamepad {
		popupSprite.Image = composePopupImage(popup.Base, popup.KeyboardCue)
		if usingGamepad {
			popupSprite.Image = composePopupImage(popup.Base, popup.GamepadCue)
		}
		popup.HasRenderedImage = true
		popup.RenderedGamepad = usingGamepad
	}

	if popupSprite.Image != nil && popupSprite.OriginX == 0 && popupSprite.OriginY == 0 {
		imageBounds := popupSprite.Image.Bounds()
		if imageBounds.Dx() > 0 && imageBounds.Dy() > 0 {
			popupSprite.OriginX = float64(imageBounds.Dx()) / 2
			popupSprite.OriginY = float64(imageBounds.Dy())
		}
	}

	popup.TargetTransitionEntity = uint64(transitionEntity)
	popupSprite.Disabled = false
	popupTransform.X = bounds.x + bounds.w/2
	popupTransform.Y = bounds.y - transitionPopupVerticalGap
	popupTransform.Rotation = 0
	if popupTransform.ScaleX == 0 {
		popupTransform.ScaleX = 1
	}
	if popupTransform.ScaleY == 0 {
		popupTransform.ScaleY = 1
	}
}

func hideTransitionPopup(popup *component.TransitionPopup, popupSprite *component.Sprite) {
	if popup == nil || popupSprite == nil {
		return
	}
	popup.TargetTransitionEntity = 0
	popupSprite.Disabled = true
}

func playerBoundsForInsideTransition(w *ecs.World) (aabb, bool) {
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return aabb{}, false
	}
	return playerAABB(w, player)
}

func activeInsideTransition(w *ecs.World, playerAABB aabb) (ecs.Entity, *component.Transition, aabb, bool) {
	bestEntity := ecs.Entity(0)
	var bestTransition *component.Transition
	bestBounds := aabb{}
	bestDistance := math.MaxFloat64
	found := false
	playerCenterX := playerAABB.x + playerAABB.w/2
	playerCenterY := playerAABB.y + playerAABB.h/2

	ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, tr *component.Transition, _ *component.Transform) {
		if tr == nil || tr.TargetLevel == "" || tr.LinkedID == "" || component.NormalizeTransitionType(tr.Type) != component.TransitionTypeInside {
			return
		}

		bounds := transitionAABB(w, e, tr)
		if !aabbIntersects(playerAABB, bounds) {
			return
		}

		centerX := bounds.x + bounds.w/2
		centerY := bounds.y + bounds.h/2
		dx := centerX - playerCenterX
		dy := centerY - playerCenterY
		distance := dx*dx + dy*dy
		if found && distance >= bestDistance {
			return
		}

		found = true
		bestDistance = distance
		bestEntity = e
		bestTransition = tr
		bestBounds = bounds
	})

	return bestEntity, bestTransition, bestBounds, found
}
