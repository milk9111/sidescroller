package system

import (
	"image/color"
	"math"

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
	inputComp, ok := ecs.Get(w, player, component.InputComponent)
	if !ok {
		inputComp = component.Input{}
	}

	isAiming := stateComp.State.Name() == "aim"

	sprite, ok := ecs.Get(w, a.aimTargetEntity, component.SpriteComponent)
	if !ok {
		return
	}
	line, ok := ecs.Get(w, a.aimTargetEntity, component.LineRenderComponent)
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
		if line.Width != 0 {
			line.Width = 0
			if err := ecs.Add(w, a.aimTargetEntity, component.LineRenderComponent, line); err != nil {
				panic("aim system: update line: " + err.Error())
			}
		}
		return
	}

	transform, ok := ecs.Get(w, a.aimTargetEntity, component.TransformComponent)
	if !ok {
		transform = component.Transform{ScaleX: 1, ScaleY: 1}
	}

	playerTransform, ok := ecs.Get(w, player, component.TransformComponent)
	if !ok {
		return
	}
	playerSprite, ok := ecs.Get(w, player, component.SpriteComponent)
	if !ok || playerSprite.Image == nil {
		return
	}
	img := playerSprite.Image
	if playerSprite.UseSource {
		if sub, ok := playerSprite.Image.SubImage(playerSprite.Source).(*ebiten.Image); ok {
			img = sub
		}
	}
	imgW := float64(img.Bounds().Dx())
	imgH := float64(img.Bounds().Dy())
	scaleX := playerTransform.ScaleX
	if scaleX == 0 {
		scaleX = 1
	}
	scaleY := playerTransform.ScaleY
	if scaleY == 0 {
		scaleY = 1
	}
	startX := playerTransform.X - playerSprite.OriginX*scaleX + (imgW*scaleX)/2
	startY := playerTransform.Y - playerSprite.OriginY*scaleY + (imgH*scaleY)/2

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

	const aimStickDeadzone = 0.2
	// aimCursorDistance controls how far the right-stick aims from the player.
	// Lowering this reduces gamepad aim sensitivity.
	const aimCursorDistance = 100.0
	useGamepadAim := inputComp.Aim && math.Hypot(inputComp.AimX, inputComp.AimY) > aimStickDeadzone
	var cursorWorldX, cursorWorldY float64
	if useGamepadAim {
		len := math.Hypot(inputComp.AimX, inputComp.AimY)
		dirX := inputComp.AimX / len
		dirY := inputComp.AimY / len
		cursorWorldX = startX + dirX*aimCursorDistance
		cursorWorldY = startY + dirY*aimCursorDistance
	} else {
		sx, sy := ebiten.CursorPosition()
		cursorWorldX = camX + float64(sx)/zoom
		cursorWorldY = camY + float64(sy)/zoom
	}

	endWorldX := cursorWorldX
	endWorldY := cursorWorldY
	dirX := cursorWorldX - startX
	dirY := cursorWorldY - startY
	len := math.Hypot(dirX, dirY)
	if len > 0 {
		dirX /= len
		dirY /= len
		maxDist := 10000.0
		if boundsEntity, ok := w.First(component.LevelBoundsComponent.Kind()); ok {
			if bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent); ok {
				if bounds.Width > 0 || bounds.Height > 0 {
					maxDist = math.Hypot(bounds.Width, bounds.Height) * 2
				}
			}

		}
		farX := startX + dirX*maxDist
		farY := startY + dirY*maxDist
		endWorldX = farX
		endWorldY = farY
		if hitX, hitY, ok := firstStaticHit(w, player, startX, startY, farX, farY); ok {
			endWorldX = hitX
			endWorldY = hitY
		}
	}

	transform.X = cursorWorldX
	transform.Y = cursorWorldY

	if sprite.Image == nil {
		sprite.Image = a.aimTargetInvalidImage
	}

	line.StartX = startX
	line.StartY = startY
	line.EndX = endWorldX
	line.EndY = endWorldY
	if line.Width <= 0 {
		line.Width = 1
	}
	if line.Color == (color.RGBA{}) {
		line.Color = color.RGBA{R: 255, A: 255}
	}

	if err := ecs.Add(w, a.aimTargetEntity, component.TransformComponent, transform); err != nil {
		panic("aim system: update transform: " + err.Error())
	}
	if err := ecs.Add(w, a.aimTargetEntity, component.SpriteComponent, sprite); err != nil {
		panic("aim system: update sprite: " + err.Error())
	}
	if err := ecs.Add(w, a.aimTargetEntity, component.LineRenderComponent, line); err != nil {
		panic("aim system: update line: " + err.Error())
	}
}
