package system

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type CameraSystem struct {
	camEntity            ecs.Entity
	targetEntity         ecs.Entity
	screenW              float64
	screenH              float64
	initialized          bool
	shakeFramesRemaining int
	shakeTotalFrames     int
	shakeIntensity       float64
	shakeTime            float64
	lastShakeX           float64
	lastShakeY           float64
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
			cs.initialized = false
			cs.shakeFramesRemaining = 0
			cs.shakeTotalFrames = 0
			cs.shakeIntensity = 0
			cs.lastShakeX = 0
			cs.lastShakeY = 0
		}
	}

	if !cs.camEntity.Valid() || !ecs.IsAlive(w, cs.camEntity) {
		if camEntity, ok := ecs.First(w, component.CameraComponent.Kind()); ok {
			cs.camEntity = camEntity
			cs.initialized = false
			cs.shakeFramesRemaining = 0
			cs.shakeTotalFrames = 0
			cs.shakeIntensity = 0
			cs.lastShakeX = 0
			cs.lastShakeY = 0
		}
	}
	if !cs.camEntity.Valid() || !ecs.IsAlive(w, cs.camEntity) {
		return
	}

	camComp, ok := ecs.Get(w, cs.camEntity, component.CameraComponent.Kind())
	if !ok {
		return
	}

	if req, ok := ecs.Get(w, cs.camEntity, component.CameraShakeRequestComponent.Kind()); ok && req != nil {
		frames := req.Frames
		if frames <= 0 {
			frames = 8
		}
		intensity := req.Intensity
		if intensity < 0 {
			intensity = 0
		}
		if frames > cs.shakeFramesRemaining {
			cs.shakeFramesRemaining = frames
			cs.shakeTotalFrames = frames
		}
		if intensity > cs.shakeIntensity {
			cs.shakeIntensity = intensity
		}
		_ = ecs.Remove(w, cs.camEntity, component.CameraShakeRequestComponent.Kind())
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

	// Debug keys: set smoothness and print current values for diagnosis.
	if inpututil.IsKeyJustPressed(ebiten.Key1) || inpututil.IsKeyJustPressed(ebiten.Key2) || inpututil.IsKeyJustPressed(ebiten.Key3) || inpututil.IsKeyJustPressed(ebiten.Key4) || inpututil.IsKeyJustPressed(ebiten.Key5) {
		if inpututil.IsKeyJustPressed(ebiten.Key1) {
			camComp.Smoothness = 0.01
		}
		if inpututil.IsKeyJustPressed(ebiten.Key2) {
			camComp.Smoothness = 0.05
		}
		if inpututil.IsKeyJustPressed(ebiten.Key3) {
			camComp.Smoothness = 0.15
		}
		if inpututil.IsKeyJustPressed(ebiten.Key4) {
			camComp.Smoothness = 0.5
		}
		if inpututil.IsKeyJustPressed(ebiten.Key5) {
			camComp.Smoothness = 1.0
		}
		_ = ecs.Add(w, cs.camEntity, component.CameraComponent.Kind(), camComp)

		// compute dt and alpha for info
		dt := 1.0 / 60.0
		if t := ebiten.ActualTPS(); t > 0 {
			dt = 1.0 / t
		}
		var alpha float64
		if camComp.Smoothness <= 0 {
			alpha = 0
		} else if camComp.Smoothness >= 1 {
			alpha = 1
		} else {
			alpha = 1 - math.Pow(1-camComp.Smoothness, 60*dt)
		}
		// current cam transform
		if ct, ok := ecs.Get(w, cs.camEntity, component.TransformComponent.Kind()); ok {
			fmt.Printf("cam smooth=%.4f alpha=%.4f camX=%.2f centerX=%.2f dx=%.2f\n", camComp.Smoothness, alpha, ct.X, centerX, centerX-ct.X)
		} else {
			fmt.Printf("cam smooth=%.4f alpha=%.4f centerX=%.2f\n", camComp.Smoothness, alpha, centerX)
		}
	}

	// Smoothly interpolate the camera transform toward the desired center.
	// `Smoothness` in prefabs is a per-frame factor in [0,1] (1 = instant).
	// Interpolating with the raw per-frame factor makes the behavior
	// framerate-dependent. Convert it to a framerate-independent exponential
	// smoothing alpha so the same Smoothness value behaves consistently.
	smooth := 1.0
	if camComp != nil {
		if camComp.Smoothness > 0 && camComp.Smoothness <= 1 {
			smooth = camComp.Smoothness
		}
	}
	if camTransform, ok := ecs.Get(w, cs.camEntity, component.TransformComponent.Kind()); ok {
		// Remove previously applied shake offset before follow interpolation
		// so shake does not accumulate into the smoothed camera position.
		camTransform.X -= cs.lastShakeX
		camTransform.Y -= cs.lastShakeY
		cs.lastShakeX = 0
		cs.lastShakeY = 0

		// If the level was just loaded, snap immediately to the target center,
		// but only the first time after the camera entity is initialized. This
		// avoids repeatedly snapping while `LevelLoadedComponent` may remain
		// present across frames.
		if !cs.initialized {
			if _, loaded := ecs.First(w, component.LevelLoadedComponent.Kind()); loaded {
				camTransform.X = centerX
				camTransform.Y = centerY
				if err := ecs.Add(w, cs.camEntity, component.TransformComponent.Kind(), camTransform); err != nil {
					panic("camera system: update transform: " + err.Error())
				}
				cs.initialized = true
				return
			}
		}

		// Compute frame delta time from ebiten's current TPS; fallback to 60 TPS.
		dt := 1.0
		if t := ebiten.ActualTPS(); t > 0 {
			dt = 1.0 / t
		} else {
			dt = 1.0 / 60.0
		}

		// Map the per-frame smooth factor `smooth` to a framerate-independent
		// alpha. At the reference 60 TPS this produces the same per-frame
		// interpolation as using `smooth` directly.
		var alpha float64
		if smooth <= 0 {
			alpha = 0
		} else if smooth >= 1 {
			alpha = 1
		} else {
			alpha = 1 - math.Pow(1-smooth, 60*dt)
		}

		camTransform.X = camTransform.X + (centerX-camTransform.X)*alpha
		camTransform.Y = camTransform.Y + (centerY-camTransform.Y)*alpha

		if cs.shakeFramesRemaining > 0 && cs.shakeIntensity > 0 {
			decay := 1.0
			if cs.shakeTotalFrames > 0 {
				decay = float64(cs.shakeFramesRemaining) / float64(cs.shakeTotalFrames)
			}
			amp := cs.shakeIntensity * decay
			cs.shakeTime += 1
			shakeX := math.Sin(cs.shakeTime*2.7) * amp
			shakeY := math.Cos(cs.shakeTime*3.9) * amp
			camTransform.X += shakeX
			camTransform.Y += shakeY
			cs.lastShakeX = shakeX
			cs.lastShakeY = shakeY
			cs.shakeFramesRemaining--
			if cs.shakeFramesRemaining <= 0 {
				cs.shakeTotalFrames = 0
				cs.shakeIntensity = 0
			}
		}

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
