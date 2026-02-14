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
	screenW      float64
	screenH      float64
}

func NewCameraSystem() *CameraSystem {
	return &CameraSystem{}
}

// SetScreenSize updates the screen dimensions used for view calculations.
// Call this each frame with the actual game screen size from LayoutF.
func (cs *CameraSystem) SetScreenSize(w, h float64) {
	cs.screenW = w
	cs.screenH = h
}

// Update sets the camera entity's transform to the target entity's position.
func (cs *CameraSystem) Update(w *ecs.World) {
	// The world is recreated on level transitions. Entity IDs can be reused across
	// worlds, so a cached entity may still be "alive" but refer to the wrong thing.
	// Validate required components before trusting cached entities.
	if cs.camEntity.Valid() && ecs.IsAlive(w, cs.camEntity) {
		if !ecs.Has(w, cs.camEntity, component.CameraComponent.Kind()) || !ecs.Has(w, cs.camEntity, component.TransformComponent.Kind()) {
			cs.camEntity = 0
		}
	}

	if !cs.camEntity.Valid() || !ecs.IsAlive(w, cs.camEntity) {
		if camEntity, ok := ecs.First(w, component.CameraComponent.Kind()); ok {
			cs.camEntity = camEntity
		}
	}
	if !cs.camEntity.Valid() || !ecs.IsAlive(w, cs.camEntity) {
		return
	}

	camComp, ok := ecs.Get(w, cs.camEntity, component.CameraComponent.Kind())
	if !ok {
		return
	}

	if cs.targetEntity.Valid() && ecs.IsAlive(w, cs.targetEntity) {
		if !ecs.Has(w, cs.targetEntity, component.TransformComponent.Kind()) {
			cs.targetEntity = 0
		} else if camComp.TargetName == "player" && !ecs.Has(w, cs.targetEntity, component.PlayerTagComponent.Kind()) {
			cs.targetEntity = 0
		}
	}

	if !cs.targetEntity.Valid() || !ecs.IsAlive(w, cs.targetEntity) {
		targetEntity := findEntityByNameOrTag(w, camComp.TargetName)
		if targetEntity.Valid() {
			cs.targetEntity = targetEntity
		}
	}

	targetTransform, ok := ecs.Get(w, cs.targetEntity, component.TransformComponent.Kind())
	if !ok {
		return
	}

	// Get the sprite size and origin for centering
	sprite, hasSprite := ecs.Get(w, cs.targetEntity, component.SpriteComponent.Kind())
	imgW, imgH := 0.0, 0.0
	if hasSprite && sprite.Image != nil {
		w := sprite.Image.Bounds().Dx()
		h := sprite.Image.Bounds().Dy()
		imgW = float64(w)
		imgH = float64(h)
	}

	sw, sh := cs.screenW, cs.screenH
	if sw <= 0 || sh <= 0 {
		// Fallback if screen size hasn't been set yet
		mw, mh := ebiten.Monitor().Size()
		sw, sh = float64(mw), float64(mh)
	}
	zoom := 1.0
	if camComp, ok := ecs.Get(w, cs.camEntity, component.CameraComponent.Kind()); ok {
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

	viewW := sw / zoom
	viewH := sh / zoom
	halfW := viewW / 2.0
	halfH := viewH / 2.0
	centerX := visualCenterX
	centerY := visualCenterY

	// Clamp to level bounds if available (match example logic)
	if boundsEntity, ok := ecs.First(w, component.LevelBoundsComponent.Kind()); ok {
		if bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent.Kind()); ok {
			if bounds.Width > 0 {
				minX := halfW
				maxX := bounds.Width - halfW
				if maxX < minX {
					centerX = bounds.Width / 2.0
				} else {
					centerX = math.Max(minX, math.Min(centerX, maxX))
				}
			}

			if bounds.Height > 0 {
				minY := halfH
				maxY := bounds.Height - halfH
				if maxY < minY {
					centerY = bounds.Height / 2.0
				} else {
					centerY = math.Max(minY, math.Min(centerY, maxY))
				}
			}
		}
	}

	// Convert camera center to top-left for rendering
	centerX -= halfW
	centerY -= halfH

	// Smoothly interpolate the camera transform toward the desired center
	smooth := 1.0
	if camComp != nil {
		if camComp.Smoothness > 0 && camComp.Smoothness <= 1 {
			smooth = camComp.Smoothness
		}
	}
	if camTransform, ok := ecs.Get(w, cs.camEntity, component.TransformComponent.Kind()); ok {
		// If the level was just loaded, snap immediately to the target center
		if _, loaded := ecs.First(w, component.LevelLoadedComponent.Kind()); loaded {
			camTransform.X = centerX
			camTransform.Y = centerY
			if err := ecs.Add(w, cs.camEntity, component.TransformComponent.Kind(), camTransform); err != nil {
				panic("camera system: update transform: " + err.Error())
			}
			return
		}

		// Lerp from current to target by smooth factor (0 = no movement, 1 = instant)
		camTransform.X = camTransform.X + (centerX-camTransform.X)*smooth
		camTransform.Y = camTransform.Y + (centerY-camTransform.Y)*smooth
		if err := ecs.Add(w, cs.camEntity, component.TransformComponent.Kind(), camTransform); err != nil {
			panic("camera system: update transform: " + err.Error())
		}
	}
}

func findEntityByNameOrTag(w *ecs.World, name string) ecs.Entity {
	if name == "player" {
		if e, ok := ecs.First(w, component.PlayerTagComponent.Kind()); ok {
			return e
		}
	}
	return 0
}
