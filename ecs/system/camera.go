package system

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type CameraSystem struct {
	camEntity    ecs.Entity
	targetEntity ecs.Entity
}

func NewCameraSystem() *CameraSystem {
	return &CameraSystem{}
}

// Update sets the camera entity's transform to the target entity's position.
func (cs *CameraSystem) Update(w *ecs.World) {
	if !cs.camEntity.Valid() || !w.IsAlive(cs.camEntity) {
		if camEntity, ok := w.First(component.CameraComponent.Kind()); ok {
			cs.camEntity = camEntity
		}
	}

	if !cs.targetEntity.Valid() || !w.IsAlive(cs.targetEntity) {
		camComp, ok := ecs.Get(w, cs.camEntity, component.CameraComponent)
		if !ok {
			return
		}

		targetEntity := findEntityByNameOrTag(w, camComp.TargetName)
		if targetEntity.Valid() {
			cs.targetEntity = targetEntity
		}
	}

	targetTransform, ok := ecs.Get(w, cs.targetEntity, component.TransformComponent)
	if !ok {
		return
	}

	// Get the sprite size and origin for centering
	sprite, hasSprite := ecs.Get(w, cs.targetEntity, component.SpriteComponent)
	imgW, imgH := 0.0, 0.0
	if hasSprite && sprite.Image != nil {
		w := sprite.Image.Bounds().Dx()
		h := sprite.Image.Bounds().Dy()
		imgW = float64(w)
		imgH = float64(h)
	}

	sw, sh := ebiten.Monitor().Size()
	zoom := 1.0
	if camComp, ok := ecs.Get(w, cs.camEntity, component.CameraComponent); ok {
		if camComp.Zoom > 0 {
			zoom = camComp.Zoom
		}
	}
	scaleX := targetTransform.ScaleX
	if scaleX == 0 {
		scaleX = 1
	}
	scaleY := targetTransform.ScaleY
	if scaleY == 0 {
		scaleY = 1
	}

	// Visual center in world coordinates
	visualCenterX := targetTransform.X - sprite.OriginX*scaleX + (imgW*scaleX)/2
	visualCenterY := targetTransform.Y - sprite.OriginY*scaleY + (imgH*scaleY)/2

	halfW := float64(sw) / (2 * zoom)
	halfH := float64(sh) / (2 * zoom)
	centerX := visualCenterX - halfW
	centerY := visualCenterY - halfH

	// Clamp to level bounds if available
	if boundsEntity, ok := w.First(component.LevelBoundsComponent.Kind()); ok {
		if bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent); ok {
			maxX := bounds.Width - float64(sw)/zoom
			maxY := bounds.Height - float64(sh)/zoom
			if maxX < 0 {
				maxX = 0
			}
			if maxY < 0 {
				maxY = 0
			}
			centerX = math.Max(0, math.Min(centerX, maxX))
			centerY = math.Max(0, math.Min(centerY, maxY))
		}
	}
	if camTransform, ok := ecs.Get(w, cs.camEntity, component.TransformComponent); ok {
		camTransform.X = centerX
		camTransform.Y = centerY
		if err := ecs.Add(w, cs.camEntity, component.TransformComponent, camTransform); err != nil {
			panic("camera system: update transform: " + err.Error())
		}
	}
}

func findEntityByNameOrTag(w *ecs.World, name string) ecs.Entity {
	if name == "player" {
		if e, ok := w.First(component.PlayerTagComponent.Kind()); ok {
			return e
		}
	}
	return 0
}
