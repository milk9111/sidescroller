package system

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AimSystem struct {
	aimTargetEntity ecs.Entity
	camEntity       ecs.Entity

	aimTargetValidImage   *ebiten.Image
	aimTargetInvalidImage *ebiten.Image
}

func NewAimSystem() *AimSystem {
	return &AimSystem{}
}

func (a *AimSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	if a.aimTargetValidImage == nil {
		img, err := assets.LoadImage("aim_target.png")
		if err != nil {
			return
		}
		a.aimTargetValidImage = img
	}

	if a.aimTargetInvalidImage == nil {
		img, err := assets.LoadImage("aim_target_invalid.png")
		if err != nil {
			return
		}
		a.aimTargetInvalidImage = img
	}

	if !a.aimTargetEntity.Valid() || !w.IsAlive(a.aimTargetEntity) {
		if aimEntity, ok := w.First(component.AimTargetTagComponent.Kind()); ok {
			a.aimTargetEntity = aimEntity
		}
	}

	if !a.aimTargetEntity.Valid() || !w.IsAlive(a.aimTargetEntity) {
		return
	}

	if !a.camEntity.Valid() || !w.IsAlive(a.camEntity) {
		if camEntity, ok := w.First(component.CameraComponent.Kind()); ok {
			a.camEntity = camEntity
		}
	}

	player, ok := w.First(component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	stateComp, ok := ecs.Get(w, player, component.PlayerStateMachineComponent)
	if !ok || stateComp.State == nil {
		return
	}

	isAiming := stateComp.State.Name() == "aim"

	sprite, ok := ecs.Get(w, a.aimTargetEntity, component.SpriteComponent)
	if !ok {
		return
	}

	if !isAiming {
		if sprite.Image != nil {
			sprite.Image = nil
			if err := ecs.Add(w, a.aimTargetEntity, component.SpriteComponent, sprite); err != nil {
				panic("aim system: update sprite: " + err.Error())
			}
		}
		return
	}

	transform, ok := ecs.Get(w, a.aimTargetEntity, component.TransformComponent)
	if !ok {
		transform = component.Transform{ScaleX: 1, ScaleY: 1}
	}

	camX, camY := 0.0, 0.0
	zoom := 1.0
	if a.camEntity.Valid() {
		if camTransform, ok := ecs.Get(w, a.camEntity, component.TransformComponent); ok {
			camX = camTransform.X
			camY = camTransform.Y
		}
		if camComp, ok := ecs.Get(w, a.camEntity, component.CameraComponent); ok {
			if camComp.Zoom > 0 {
				zoom = camComp.Zoom
			}
		}
	}

	sx, sy := ebiten.CursorPosition()
	transform.X = camX + float64(sx)/zoom
	transform.Y = camY + float64(sy)/zoom

	if sprite.Image == nil {
		sprite.Image = a.aimTargetInvalidImage
	}

	if err := ecs.Add(w, a.aimTargetEntity, component.TransformComponent, transform); err != nil {
		panic("aim system: update transform: " + err.Error())
	}
	if err := ecs.Add(w, a.aimTargetEntity, component.SpriteComponent, sprite); err != nil {
		panic("aim system: update sprite: " + err.Error())
	}
}
