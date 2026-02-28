package system

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type PickupCollectSystem struct{}

const (
	anchorTutorialText       = "Hold LT to aim anchor. Press RT while aiming to shoot anchor."
	anchorTutorialFrames     = 10 * 60
	anchorTutorialLayer      = 1100
	anchorTutorialPaddingX   = 8
	anchorTutorialPaddingY   = 4
	anchorTutorialPromptW    = 420
	anchorTutorialPromptH    = 24
	anchorTutorialPromptTopY = 44.0
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

	px := aabbTopLeftX(w, player, playerTransform.X, playerBody.OffsetX, playerBody.Width, playerBody.AlignTopLeft)
	py := playerTransform.Y + playerBody.OffsetY
	if !playerBody.AlignTopLeft {
		py -= playerBody.Height / 2
	}
	pw := playerBody.Width
	ph := playerBody.Height

	ecs.ForEach2(w, component.PickupComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, pickup *component.Pickup, t *component.Transform) {
		if pickup == nil || t == nil {
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
		if !intersects(px, py, pw, ph, kx, ky, kw, kh) {
			return
		}

		if audioComp, ok := ecs.Get(w, e, component.AudioComponent.Kind()); ok && audioComp != nil {
			for i, name := range audioComp.Names {
				if name != "pickup" {
					continue
				}
				if i < len(audioComp.Play) {
					audioComp.Play[i] = true
				}
				break
			}
			_ = ecs.Add(w, e, component.AudioComponent.Kind(), audioComp)
		}

		if abilitiesEntity, found := ecs.First(w, component.AbilitiesComponent.Kind()); found {
			if abilities, ok := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind()); ok && abilities != nil {
				if pickup.GrantDoubleJump {
					abilities.DoubleJump = true
				}
				if pickup.GrantWallGrab {
					abilities.WallGrab = true
				}
				if pickup.GrantAnchor {
					abilities.Anchor = true
					showAnchorTutorialHint(w)
				}
				_ = ecs.Add(w, abilitiesEntity, component.AbilitiesComponent.Kind(), abilities)
			}
		} else {
			ent := ecs.CreateEntity(w)
			_ = ecs.Add(w, ent, component.AbilitiesComponent.Kind(), &component.Abilities{
				DoubleJump: pickup.GrantDoubleJump,
				WallGrab:   pickup.GrantWallGrab,
				Anchor:     pickup.GrantAnchor,
			})
			if pickup.GrantAnchor {
				showAnchorTutorialHint(w)
			}
		}

		if pickup.Kind == "trophy" {
			if trackerEntity, found := ecs.First(w, component.TrophyTrackerComponent.Kind()); found {
				if tracker, ok := ecs.Get(w, trackerEntity, component.TrophyTrackerComponent.Kind()); ok && tracker != nil {
					tracker.Count++
					_ = ecs.Add(w, trackerEntity, component.TrophyTrackerComponent.Kind(), tracker)
				}
			}
		}

		// AudioSystem runs before PickupCollectSystem in the scheduler. If we
		// destroy immediately, queued pickup audio never gets processed.
		// Remove pickup behavior now, hide sprite, and destroy shortly after.
		_ = ecs.Remove(w, e, component.PickupComponent.Kind())
		_ = ecs.Remove(w, e, component.SpriteComponent.Kind())
		_ = ecs.Add(w, e, component.TTLComponent.Kind(), &component.TTL{Frames: 2})
	})
}

func showAnchorTutorialHint(w *ecs.World) {
	if w == nil {
		return
	}

	img := ebiten.NewImage(anchorTutorialPromptW, anchorTutorialPromptH)
	img.Fill(color.NRGBA{R: 0, G: 0, B: 0, A: 170})
	ebitenutil.DebugPrintAt(img, anchorTutorialText, anchorTutorialPaddingX, anchorTutorialPaddingY)

	screenW, _ := ebiten.WindowSize()
	if screenW <= 0 {
		monitorW, _ := ebiten.Monitor().Size()
		screenW = monitorW
	}
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
