package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
)

type PickupCollectSystem struct{}

const (
	anchorTutorialTextGamepad  = "Hold LT to aim anchor. Press RT while aiming to shoot anchor.\nPress RT without aiming to shoot an automatic anchor.\nHold X/Y to reel in/out.\nPress RT to release."
	anchorTutorialTextKeyboard = "Hold RMB to aim anchor. Press LMB while aiming to shoot anchor.\nPress Ctrl without aiming to shoot an automatic anchor.\nHold Q/E to reel in/out.\nPress Space to release."
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

	text := anchorTutorialTextKeyboard

	inputEnt, ok := ecs.First(w, component.InputComponent.Kind())
	if ok {
		if input, ok := ecs.Get(w, inputEnt, component.InputComponent.Kind()); ok && input != nil {
			if input.UsingGamepad {
				text = anchorTutorialTextGamepad
			}
		}
	}
	_ = entity.ShowTimedDebugMessage(w, entity.DebugMessageDefaultWidth, entity.DebugMessageDefaultHeight, text, entity.DebugMessageDefaultFrames)
}
