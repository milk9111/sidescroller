package system

import (
	"math"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type ParallaxSystem struct{}

func NewParallaxSystem() *ParallaxSystem { return &ParallaxSystem{} }

func (s *ParallaxSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	camEntity, ok := ecs.First(w, component.CameraComponent.Kind())
	if !ok {
		return
	}

	camTransform, ok := ecs.Get(w, camEntity, component.TransformComponent.Kind())
	if !ok || camTransform == nil {
		return
	}

	camX, camY := cameraPosition(camTransform)
	zoom := 1.0
	if camComp, ok := ecs.Get(w, camEntity, component.CameraComponent.Kind()); ok && camComp != nil && camComp.Zoom > 0 {
		zoom = camComp.Zoom
	}
	bounds := activeLevelBounds(w)

	ecs.ForEach2(w, component.ParallaxComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, parallax *component.Parallax, t *component.Transform) {
		if parallax == nil || t == nil {
			return
		}

		if !parallax.Initialized {
			var sprite *component.Sprite
			if existing, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok {
				sprite = existing
			}

			refCamX, refCamY := parallaxReferenceCameraPosition(t, sprite, bounds, zoom)
			parallax.BaseX = t.X
			parallax.BaseY = t.Y
			parallax.CameraBaseX = refCamX
			parallax.CameraBaseY = refCamY
			if parallax.HasAnchorCameraX {
				parallax.CameraBaseX = parallax.AnchorCameraX
			}
			if parallax.HasAnchorCameraY {
				parallax.CameraBaseY = parallax.AnchorCameraY
			}
			parallax.Initialized = true
		}

		dx := camX - parallax.CameraBaseX
		dy := camY - parallax.CameraBaseY
		t.X = parallax.BaseX + dx*parallax.FactorX
		t.Y = parallax.BaseY + dy*parallax.FactorY
	})
}

func parallaxReferenceCameraPosition(t *component.Transform, sprite *component.Sprite, bounds *component.LevelBounds, zoom float64) (float64, float64) {
	if t == nil {
		return 0, 0
	}
	if zoom <= 0 {
		zoom = 1
	}

	viewW := common.BaseWidth / zoom
	viewH := common.BaseHeight / zoom
	halfW := viewW / 2.0
	halfH := viewH / 2.0

	centerX, centerY := parallaxVisualCenter(t, sprite)
	centerX = clampCameraCenter(centerX, halfW, boundsWidth(bounds))
	centerY = clampCameraCenter(centerY, halfH, boundsHeight(bounds))

	return centerX - halfW, centerY - halfH
}

func parallaxVisualCenter(t *component.Transform, sprite *component.Sprite) (float64, float64) {
	if t == nil {
		return 0, 0
	}

	centerX := t.X
	centerY := t.Y
	if sprite == nil || sprite.Image == nil {
		return centerX, centerY
	}

	scaleX := t.ScaleX
	if scaleX == 0 {
		scaleX = 1
	}
	scaleY := t.ScaleY
	if scaleY == 0 {
		scaleY = 1
	}

	imgW := float64(sprite.Image.Bounds().Dx())
	imgH := float64(sprite.Image.Bounds().Dy())
	if sprite.UseSource {
		srcW := sprite.Source.Dx()
		srcH := sprite.Source.Dy()
		if srcW > 0 {
			imgW = float64(srcW)
		}
		if srcH > 0 {
			imgH = float64(srcH)
		}
	}

	centerX = t.X - sprite.OriginX*scaleX + (imgW*scaleX)/2
	centerY = t.Y - sprite.OriginY*scaleY + (imgH*scaleY)/2
	return centerX, centerY
}

func clampCameraCenter(center, halfView, worldSize float64) float64 {
	if worldSize <= 0 {
		return center
	}
	minCenter := halfView
	maxCenter := worldSize - halfView
	if maxCenter < minCenter {
		return worldSize / 2.0
	}
	return math.Max(minCenter, math.Min(center, maxCenter))
}

func boundsWidth(bounds *component.LevelBounds) float64 {
	if bounds == nil {
		return 0
	}
	return bounds.Width
}

func boundsHeight(bounds *component.LevelBounds) float64 {
	if bounds == nil {
		return 0
	}
	return bounds.Height
}

func cameraPosition(t *component.Transform) (float64, float64) {
	if t == nil {
		return 0, 0
	}
	if t.Parent != 0 {
		return t.WorldX, t.WorldY
	}
	return t.X, t.Y
}
