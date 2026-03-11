package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func entityFacingLeft(w *ecs.World, e ecs.Entity) bool {
	if w == nil || !e.Valid() {
		return false
	}
	s, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
	return ok && s != nil && s.FacingLeft
}

func entitySpriteWidth(w *ecs.World, e ecs.Entity) (float64, bool) {
	if w == nil || !e.Valid() {
		return 0, false
	}
	s, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
	if !ok || s == nil || s.Image == nil {
		return 0, false
	}
	if s.UseSource {
		srcW := s.Source.Dx()
		if srcW > 0 {
			return float64(srcW), true
		}
	}
	wid := s.Image.Bounds().Dx()
	if wid <= 0 {
		return 0, false
	}
	return float64(wid), true
}

func facingAdjustedOffsetX(w *ecs.World, e ecs.Entity, offsetX, aabbWidth float64, alignTopLeft bool) float64 {
	if !entityFacingLeft(w, e) {
		return offsetX
	}
	if spriteW, ok := entitySpriteWidth(w, e); ok && spriteW > 0 {
		if alignTopLeft {
			return spriteW - offsetX - aabbWidth
		}
		return spriteW - offsetX
	}
	if alignTopLeft {
		return -offsetX - aabbWidth
	}
	return -offsetX
}

func aabbTopLeftX(w *ecs.World, e ecs.Entity, transformX, offsetX, aabbWidth float64, alignTopLeft bool) float64 {
	effectiveOffsetX := facingAdjustedOffsetX(w, e, offsetX, aabbWidth, alignTopLeft)
	if alignTopLeft {
		return transformX + effectiveOffsetX
	}
	return transformX + effectiveOffsetX - aabbWidth/2
}

func aabbTopLeftY(transformY, offsetY, aabbHeight float64, alignTopLeft bool) float64 {
	if alignTopLeft {
		return transformY + offsetY
	}
	return transformY + offsetY - aabbHeight/2
}

func bodyCenterX(w *ecs.World, e ecs.Entity, t *component.Transform, body *component.PhysicsBody) float64 {
	if t == nil || body == nil {
		return 0
	}
	effectiveOffsetX := facingAdjustedOffsetX(w, e, body.OffsetX, body.Width, body.AlignTopLeft)
	centerX := t.X + effectiveOffsetX
	if body.AlignTopLeft {
		centerX += body.Width / 2
	}
	return centerX
}

func bodyCenterY(t *component.Transform, body *component.PhysicsBody) float64 {
	if t == nil || body == nil {
		return 0
	}
	return aabbTopLeftY(t.Y, body.OffsetY, body.Height, body.AlignTopLeft) + body.Height/2
}

func physicsBodyBounds(w *ecs.World, e ecs.Entity, t *component.Transform, body *component.PhysicsBody) (minX, minY, maxX, maxY float64, ok bool) {
	if t == nil || body == nil {
		return 0, 0, 0, 0, false
	}
	width := body.Width
	height := body.Height
	if width <= 0 {
		width = 32
	}
	if height <= 0 {
		height = 32
	}
	minX = aabbTopLeftX(w, e, t.X, body.OffsetX, width, body.AlignTopLeft)
	minY = aabbTopLeftY(t.Y, body.OffsetY, height, body.AlignTopLeft)
	return minX, minY, minX + width, minY + height, true
}

func physicsBodyCenter(w *ecs.World, e ecs.Entity, t *component.Transform, body *component.PhysicsBody) (float64, float64, bool) {
	if body == nil {
		return 0, 0, false
	}
	if body.Body != nil {
		pos := body.Body.Position()
		return pos.X, pos.Y, true
	}
	if t == nil {
		return 0, 0, false
	}
	return bodyCenterX(w, e, t, body), bodyCenterY(t, body), true
}

func spriteBodyPivotLocal(w *ecs.World, e ecs.Entity, sprite *component.Sprite, body *component.PhysicsBody) (float64, float64, bool) {
	if sprite == nil || body == nil {
		return 0, 0, false
	}
	pivotX := sprite.OriginX + facingAdjustedOffsetX(w, e, body.OffsetX, body.Width, body.AlignTopLeft)
	pivotY := sprite.OriginY + body.OffsetY
	if body.AlignTopLeft {
		pivotX += body.Width / 2
		pivotY += body.Height / 2
	}
	return pivotX, pivotY, true
}
