package system

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type PickupCollectSystem struct{}

const (
	anchorTutorialTextGamepad  = "Hold LT to aim anchor. Press RT while aiming to shoot anchor.\nPress RT without aiming to shoot an automatic anchor.\nHold X/Y to reel in/out.\nPress RT to release."
	anchorTutorialTextKeyboard = "Hold RMB to aim anchor. Press LMB while aiming to shoot anchor.\nPress Ctrl without aiming to shoot an automatic anchor.\nHold Q/E to reel in/out.\nPress Space to release."
	anchorTutorialFrames       = 30 * 60
	anchorTutorialLayer        = 1100
	anchorTutorialPaddingX     = 8
	anchorTutorialPaddingY     = 4
	anchorTutorialPromptW      = 420
	anchorTutorialPromptH      = 96
	anchorTutorialPromptTopY   = 44.0
)

func NewPickupCollectSystem() *PickupCollectSystem { return &PickupCollectSystem{} }

func (s *PickupCollectSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	playerTransform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || playerTransform == nil {
		return
	}

	playerBody, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind())
	if !ok || playerBody == nil {
		return
	}

	px, py, pxMax, pyMax, ok := physicsBodyBounds(w, player, playerTransform, playerBody)
	if !ok {
		return
	}
	pw := pxMax - px
	ph := pyMax - py

	ecs.ForEach2(w, component.PickupComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, pickup *component.Pickup, t *component.Transform) {
		if pickup == nil || t == nil {
			return
		}

		if _, ok := ecs.Get(w, e, component.ItemComponent.Kind()); ok {
			return
		}
		if _, ok := ecs.Get(w, e, component.ItemReferenceComponent.Kind()); ok {
			return
		}

		kw := pickup.CollisionWidth
		kh := pickup.CollisionHeight
		if kw <= 0 || kh <= 0 {
			kw = 24
			kh = 24
		}

		kx := t.X
		ky := t.Y
		if _, _, hit := intersects(px, py, pw, ph, kx, ky, kw, kh); !hit {
			return
		}

		// if audioComp, ok := ecs.Get(w, e, component.AudioComponent.Kind()); ok && audioComp != nil {
		// 	for i, name := range audioComp.Names {
		// 		if name != "pickup" {
		// 			continue
		// 		}
		// 		if i < len(audioComp.Play) {
		// 			audioComp.Play[i] = true
		// 		}
		// 		break
		// 	}
		// 	_ = ecs.Add(w, e, component.AudioComponent.Kind(), audioComp)
		// }

		collectPickupEntity(w, e, pickup)
	})
}

func showAnchorTutorialHint(w *ecs.World) {
	if w == nil {
		return
	}

	img := ebiten.NewImage(anchorTutorialPromptW, anchorTutorialPromptH)
	img.Fill(color.NRGBA{R: 0, G: 0, B: 0, A: 170})

	text := anchorTutorialTextKeyboard

	inputEnt, ok := ecs.First(w, component.InputComponent.Kind())
	if ok {
		if input, ok := ecs.Get(w, inputEnt, component.InputComponent.Kind()); ok && input != nil {
			if input.UsingGamepad {
				text = anchorTutorialTextGamepad
			}
		}
	}

	ebitenutil.DebugPrintAt(img, text, anchorTutorialPaddingX, anchorTutorialPaddingY)

	screenW := common.BaseWidth
	x := float64(screenW-anchorTutorialPromptW) / 2
	if x < 8 {
		x = 8
	}

	if hintEnt, ok := ecs.First(w, component.AnchorTutorialHintComponent.Kind()); ok {
		if sprite, has := ecs.Get(w, hintEnt, component.SpriteComponent.Kind()); has && sprite != nil {
			sprite.Image = img
			_ = ecs.Add(w, hintEnt, component.SpriteComponent.Kind(), sprite)
		} else {
			_ = ecs.Add(w, hintEnt, component.SpriteComponent.Kind(), &component.Sprite{Image: img})
		}

		if transform, has := ecs.Get(w, hintEnt, component.TransformComponent.Kind()); has && transform != nil {
			transform.X = x
			transform.Y = anchorTutorialPromptTopY
			if transform.ScaleX == 0 {
				transform.ScaleX = 1
			}
			if transform.ScaleY == 0 {
				transform.ScaleY = 1
			}
			_ = ecs.Add(w, hintEnt, component.TransformComponent.Kind(), transform)
		} else {
			_ = ecs.Add(w, hintEnt, component.TransformComponent.Kind(), &component.Transform{X: x, Y: anchorTutorialPromptTopY, ScaleX: 1, ScaleY: 1})
		}

		_ = ecs.Add(w, hintEnt, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{})
		_ = ecs.Add(w, hintEnt, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: anchorTutorialLayer})
		_ = ecs.Add(w, hintEnt, component.TTLComponent.Kind(), &component.TTL{Frames: anchorTutorialFrames})
		return
	}

	hintEnt := ecs.CreateEntity(w)
	_ = ecs.Add(w, hintEnt, component.AnchorTutorialHintComponent.Kind(), &component.AnchorTutorialHint{})
	_ = ecs.Add(w, hintEnt, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{})
	_ = ecs.Add(w, hintEnt, component.TransformComponent.Kind(), &component.Transform{X: x, Y: anchorTutorialPromptTopY, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, hintEnt, component.SpriteComponent.Kind(), &component.Sprite{Image: img})
	_ = ecs.Add(w, hintEnt, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: anchorTutorialLayer})
	_ = ecs.Add(w, hintEnt, component.TTLComponent.Kind(), &component.TTL{Frames: anchorTutorialFrames})
}
