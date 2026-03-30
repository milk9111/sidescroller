package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const defaultDialogueRange = 196.0
const dialoguePopupVerticalGap = 6.0

type DialoguePopupSystem struct{}

func NewDialoguePopupSystem() *DialoguePopupSystem {
	return &DialoguePopupSystem{}
}

func (s *DialoguePopupSystem) Update(w *ecs.World) {
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

	popupEntity, ok := ecs.First(w, component.DialoguePopupComponent.Kind())
	if !ok {
		return
	}
	popup, ok := ecs.Get(w, popupEntity, component.DialoguePopupComponent.Kind())
	if !ok || popup == nil {
		return
	}

	popupSprite, ok := ecs.Get(w, popupEntity, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		return
	}

	playerX, playerY, ok := playerWorldPosition(w)
	if !ok {
		popup.TargetDialogueEntity = 0
		popupSprite.Disabled = true
		return
	}

	bestDistanceSq := math.MaxFloat64
	bestEntity := ecs.Entity(0)
	bestAnchorX := 0.0
	bestAnchorY := 0.0
	found := false

	ecs.ForEach3(w, component.DialogueComponent.Kind(), component.TransformComponent.Kind(), component.SpriteComponent.Kind(), func(e ecs.Entity, dialogue *component.Dialogue, transform *component.Transform, sprite *component.Sprite) {
		if dialogue == nil || transform == nil || sprite == nil || sprite.Image == nil || sprite.Disabled {
			return
		}

		rangeLimit := dialogue.Range
		if rangeLimit <= 0 {
			rangeLimit = defaultDialogueRange
		}

		speakerX, speakerY, ok := entityWorldPosition(w, e)
		if !ok {
			return
		}

		dx := speakerX - playerX
		dy := speakerY - playerY
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
		popup.TargetDialogueEntity = 0
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

	popup.TargetDialogueEntity = uint64(bestEntity)
	popupSprite.Disabled = false
	popupTransform.X = bestAnchorX
	popupTransform.Y = bestAnchorY - dialoguePopupVerticalGap
	popupTransform.Rotation = 0
	if popupTransform.ScaleX == 0 {
		popupTransform.ScaleX = 1
	}
	if popupTransform.ScaleY == 0 {
		popupTransform.ScaleY = 1
	}
}

func spriteTopCenterWorld(transform *component.Transform, sprite *component.Sprite) (float64, float64, bool) {
	if transform == nil || sprite == nil || sprite.Image == nil {
		return 0, 0, false
	}

	x, y, scaleX, scaleY := resolvedSpriteTransform(transform)
	width, height, ok := spriteDimensions(sprite)
	if !ok {
		return 0, 0, false
	}

	left := x - sprite.OriginX*scaleX
	top := y - sprite.OriginY*scaleY
	return left + (width*scaleX)/2, top, height > 0
}

func resolvedSpriteTransform(transform *component.Transform) (x, y, scaleX, scaleY float64) {
	if transform.Parent != 0 {
		x = transform.WorldX
		y = transform.WorldY
		scaleX = transform.WorldScaleX
		scaleY = transform.WorldScaleY
	} else {
		x = transform.X
		y = transform.Y
		scaleX = transform.ScaleX
		scaleY = transform.ScaleY
	}

	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}

	return x, y, math.Abs(scaleX), math.Abs(scaleY)
}

func spriteDimensions(sprite *component.Sprite) (float64, float64, bool) {
	if sprite == nil || sprite.Image == nil {
		return 0, 0, false
	}

	if sprite.UseSource {
		width := sprite.Source.Dx()
		height := sprite.Source.Dy()
		if width > 0 && height > 0 {
			return float64(width), float64(height), true
		}
	}

	bounds := sprite.Image.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return 0, 0, false
	}

	return float64(bounds.Dx()), float64(bounds.Dy()), true
}
