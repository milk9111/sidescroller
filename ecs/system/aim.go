package system

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
)

type AimSystem struct {
	aimTargetEntity ecs.Entity
	camEntity       ecs.Entity

	aimTargetValidImage   *ebiten.Image
	aimTargetInvalidImage *ebiten.Image
	prevAiming            bool
}

func NewAimSystem() *AimSystem { return &AimSystem{} }

func copyAnchorDrawOrder(w *ecs.World, player, anchorEnt ecs.Entity) {
	if playerLayer, ok := ecs.Get(w, player, component.EntityLayerComponent.Kind()); ok && playerLayer != nil {
		copiedLayer := *playerLayer
		if err := ecs.Add(w, anchorEnt, component.EntityLayerComponent.Kind(), &copiedLayer); err != nil {
			panic("aim system: add anchor entity layer: " + err.Error())
		}
	}
	if playerOrder, ok := ecs.Get(w, player, component.RenderLayerComponent.Kind()); ok && playerOrder != nil {
		anchorOrder := *playerOrder
		anchorOrder.Index--
		if err := ecs.Add(w, anchorEnt, component.RenderLayerComponent.Kind(), &anchorOrder); err != nil {
			panic("aim system: add anchor render order: " + err.Error())
		}
	}
}

func (a *AimSystem) fireAnchor(w *ecs.World, player ecs.Entity, startX, startY, endWorldX, endWorldY float64) {
	if w == nil {
		return
	}

	// ensure only one anchor: mark existing anchors for removal
	ecs.ForEach(w, component.AnchorTagComponent.Kind(), func(e ecs.Entity, anchorTag *component.AnchorTag) {
		if !ecs.Has(w, e, component.AnchorPendingDestroyComponent.Kind()) {
			_ = ecs.Add(w, e, component.AnchorPendingDestroyComponent.Kind(), &component.AnchorPendingDestroy{})
		}
	})

	angle := math.Atan2(endWorldY-startY, endWorldX-startX) + (math.Pi / 2)
	anchorEnt, err := entity.BuildEntity(w, "anchor.yaml")
	if err != nil {
		panic("aim system: spawn anchor: " + err.Error())
	}
	copyAnchorDrawOrder(w, player, anchorEnt)

	at, ok := ecs.Get(w, anchorEnt, component.TransformComponent.Kind())
	if !ok {
		at = &component.Transform{ScaleX: 1, ScaleY: 1}
	}
	at.X = startX
	at.Y = startY
	at.Rotation = angle
	if err := ecs.Add(w, anchorEnt, component.TransformComponent.Kind(), at); err != nil {
		panic("aim system: place anchor: " + err.Error())
	}

	anchorComp, ok := ecs.Get(w, anchorEnt, component.AnchorComponent.Kind())
	if !ok {
		panic("aim system: missing anchor component on prefab")
	}

	anchorComp.TargetX = endWorldX
	anchorComp.TargetY = endWorldY
	if err := ecs.Add(w, anchorEnt, component.AnchorComponent.Kind(), anchorComp); err != nil {
		panic("aim system: add anchor component: " + err.Error())
	}
}

func (a *AimSystem) anchorStart(w *ecs.World, player ecs.Entity) (float64, float64, bool) {
	playerTransform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok {
		return 0, 0, false
	}
	playerSprite, ok := ecs.Get(w, player, component.SpriteComponent.Kind())
	if !ok || playerSprite.Image == nil {
		return 0, 0, false
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
	return startX, startY, true
}

func (a *AimSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	// The world is recreated on level transitions. Entity IDs can be reused across
	// worlds, so a cached entity may still be "alive" but refer to the wrong thing.
	if a.aimTargetEntity.Valid() && ecs.IsAlive(w, a.aimTargetEntity) {
		if !ecs.Has(w, a.aimTargetEntity, component.AimTargetTagComponent.Kind()) {
			a.aimTargetEntity = 0
		}
	}
	if a.camEntity.Valid() && ecs.IsAlive(w, a.camEntity) {
		if !ecs.Has(w, a.camEntity, component.CameraComponent.Kind()) {
			a.camEntity = 0
		}
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	stateComp, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind())
	if !ok || stateComp.State == nil {
		return
	}
	inputComp, ok := ecs.Get(w, player, component.InputComponent.Kind())
	if !ok {
		inputComp = &component.Input{}
	}

	isAiming := stateComp.State.Name() == "aim"
	anchorAllowed := false
	if abilitiesEntity, ok := ecs.First(w, component.AbilitiesComponent.Kind()); ok {
		if abilities, ok := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind()); ok && abilities != nil {
			anchorAllowed = abilities.Anchor
		}
	}

	startX, startY, ok := a.anchorStart(w, player)
	if !ok {
		return
	}
	autoAnchorMinDistance := 0.0
	if playerComp, ok := ecs.Get(w, player, component.PlayerComponent.Kind()); ok && playerComp != nil {
		autoAnchorMinDistance = math.Max(0, playerComp.AnchorMinLength)
	}

	if !isAiming && inputComp.AutoAnchorPressed && anchorAllowed {
		if endWorldX, endWorldY, found := closestAutoAnchorTarget(w, player, startX, startY, autoAnchorMinDistance, float64(common.AnchorMaxDistance)); found {
			a.fireAnchor(w, player, startX, startY, endWorldX, endWorldY)
		}
	}

	if !a.aimTargetEntity.Valid() || !ecs.IsAlive(w, a.aimTargetEntity) {
		if aimEntity, ok := ecs.First(w, component.AimTargetTagComponent.Kind()); ok {
			a.aimTargetEntity = aimEntity
		}
	}

	if !isAiming {
		if a.aimTargetEntity.Valid() && ecs.IsAlive(w, a.aimTargetEntity) {
			if sprite, ok := ecs.Get(w, a.aimTargetEntity, component.SpriteComponent.Kind()); ok && sprite.Image != nil {
				sprite.Image = nil
				if err := ecs.Add(w, a.aimTargetEntity, component.SpriteComponent.Kind(), sprite); err != nil {
					panic("aim system: hide sprite: " + err.Error())
				}
			}
			if line, ok := ecs.Get(w, a.aimTargetEntity, component.LineRenderComponent.Kind()); ok && line.Width != 0 {
				line.Width = 0
				if err := ecs.Add(w, a.aimTargetEntity, component.LineRenderComponent.Kind(), line); err != nil {
					panic("aim system: hide line: " + err.Error())
				}
			}
		}
		a.updateAnchorRangeCircle(w, player, 0, 0, false)
		a.prevAiming = false
		return
	}

	if a.aimTargetValidImage == nil {
		img, err := assets.LoadImage("aim_target_valid.png")
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

	if !a.aimTargetEntity.Valid() || !ecs.IsAlive(w, a.aimTargetEntity) {
		return
	}

	if !a.camEntity.Valid() || !ecs.IsAlive(w, a.camEntity) {
		if camEntity, ok := ecs.First(w, component.CameraComponent.Kind()); ok {
			a.camEntity = camEntity
		}
	}

	sprite, ok := ecs.Get(w, a.aimTargetEntity, component.SpriteComponent.Kind())
	if !ok {
		return
	}
	line, ok := ecs.Get(w, a.aimTargetEntity, component.LineRenderComponent.Kind())
	if !ok {
		return
	}

	transform, ok := ecs.Get(w, a.aimTargetEntity, component.TransformComponent.Kind())
	if !ok {
		transform = &component.Transform{ScaleX: 1, ScaleY: 1}
	}
	a.updateAnchorRangeCircle(w, player, startX, startY, isAiming && anchorAllowed)

	camX, camY := 0.0, 0.0
	zoom := 1.0
	if a.camEntity.Valid() {
		if camTransform, ok := ecs.Get(w, a.camEntity, component.TransformComponent.Kind()); ok {
			camX = camTransform.X
			camY = camTransform.Y
		}
		if camComp, ok := ecs.Get(w, a.camEntity, component.CameraComponent.Kind()); ok {
			if camComp.Zoom > 0 {
				zoom = camComp.Zoom
			}
		}
	}

	const aimStickDeadzone = 0.2
	anchorMaxDistance := float64(common.AnchorMaxDistance)
	useGamepadAim := inputComp.Aim && math.Hypot(inputComp.AimX, inputComp.AimY) > aimStickDeadzone
	var cursorWorldX, cursorWorldY float64
	aimWithinRange := true
	if useGamepadAim {
		len := math.Hypot(inputComp.AimX, inputComp.AimY)
		dirX := inputComp.AimX / len
		dirY := inputComp.AimY / len
		cursorWorldX = startX + dirX*anchorMaxDistance
		cursorWorldY = startY + dirY*anchorMaxDistance
	} else {
		sx, sy := ebiten.CursorPosition()
		rawCursorWorldX := camX + float64(sx)/zoom
		rawCursorWorldY := camY + float64(sy)/zoom
		rawDX := rawCursorWorldX - startX
		rawDY := rawCursorWorldY - startY
		rawDistance := math.Hypot(rawDX, rawDY)
		if rawDistance > anchorMaxDistance {
			aimWithinRange = false
			if rawDistance > 0 {
				cursorWorldX = startX + (rawDX/rawDistance)*anchorMaxDistance
				cursorWorldY = startY + (rawDY/rawDistance)*anchorMaxDistance
			} else {
				cursorWorldX = startX
				cursorWorldY = startY
			}
		} else {
			cursorWorldX = rawCursorWorldX
			cursorWorldY = rawCursorWorldY
		}
	}

	endWorldX := cursorWorldX
	endWorldY := cursorWorldY
	hasHit := false
	anchorHitValid := false
	dirX := cursorWorldX - startX
	dirY := cursorWorldY - startY
	len := math.Hypot(dirX, dirY)
	if len > 0 && aimWithinRange {
		if hitX, hitY, ok, valid := firstStaticHit(w, player, startX, startY, cursorWorldX, cursorWorldY); ok {
			hasHit = true
			anchorHitValid = valid
			endWorldX = hitX
			endWorldY = hitY
		}
	}

	// If we just entered aiming, mark any previous anchors for removal.
	if isAiming && !a.prevAiming {
		ecs.ForEach(w, component.AnchorTagComponent.Kind(), func(e ecs.Entity, a *component.AnchorTag) {
			_ = ecs.Add(w, e, component.AnchorPendingDestroyComponent.Kind(), &component.AnchorPendingDestroy{})
		})
	}

	canFireAnchor := anchorAllowed && aimWithinRange && hasHit && anchorHitValid
	if isAiming && inputComp.AnchorPressed && canFireAnchor {
		a.fireAnchor(w, player, startX, startY, endWorldX, endWorldY)
	}

	transform.X = endWorldX
	transform.Y = endWorldY

	if canFireAnchor {
		sprite.Image = a.aimTargetValidImage
	} else {
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

	if err := ecs.Add(w, a.aimTargetEntity, component.TransformComponent.Kind(), transform); err != nil {
		panic("aim system: update transform: " + err.Error())
	}
	if err := ecs.Add(w, a.aimTargetEntity, component.SpriteComponent.Kind(), sprite); err != nil {
		panic("aim system: update sprite: " + err.Error())
	}
	if err := ecs.Add(w, a.aimTargetEntity, component.LineRenderComponent.Kind(), line); err != nil {
		panic("aim system: update line: " + err.Error())
	}

	a.prevAiming = isAiming
}

func (a *AimSystem) updateAnchorRangeCircle(w *ecs.World, player ecs.Entity, centerX, centerY float64, visible bool) {
	if w == nil || !player.Valid() || !ecs.IsAlive(w, player) {
		return
	}

	playerTransform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || playerTransform == nil {
		return
	}

	circle, ok := ecs.Get(w, player, component.CircleRenderComponent.Kind())
	if !ok || circle == nil {
		return
	}

	circle.OffsetX = centerX - playerTransform.X
	circle.OffsetY = centerY - playerTransform.Y
	circle.Radius = float64(common.AnchorMaxDistance)
	circle.Disabled = !visible
	if circle.Width <= 0 {
		circle.Width = 2
	}
	if circle.Color == nil {
		circle.Color = color.RGBA{R: 127, G: 214, B: 255, A: 255}
	}

	if err := ecs.Add(w, player, component.CircleRenderComponent.Kind(), circle); err != nil {
		panic("aim system: update anchor range circle: " + err.Error())
	}
}
