package system

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	trophyCounterPaddingX = 12.0
	trophyCounterPaddingY = 12.0
	trophyCounterSpacing  = 8.0
	trophyCounterTextW    = 64
	trophyCounterTextH    = 16
)

type TrophyCounterSystem struct{}

func NewTrophyCounterSystem() *TrophyCounterSystem { return &TrophyCounterSystem{} }

func (s *TrophyCounterSystem) Update(w *ecs.World) {
	var (
		counterEntity ecs.Entity
		trackerEntity ecs.Entity
		textEntity    ecs.Entity
		counter       *component.TrophyCounter
		tracker       *component.TrophyTracker
		iconTransform *component.Transform
		iconSprite    *component.Sprite
		textTransform *component.Transform
		textSprite    *component.Sprite
	)

	if e, ok := ecs.First(w, component.TrophyCounterComponent.Kind()); ok {
		counterEntity = e
		counter, _ = ecs.Get(w, counterEntity, component.TrophyCounterComponent.Kind())
	}

	if e, ok := ecs.First(w, component.TrophyTrackerComponent.Kind()); ok {
		trackerEntity = e
		tracker, _ = ecs.Get(w, e, component.TrophyTrackerComponent.Kind())
	}

	iconEntity, ok := ecs.First(w, component.TrophyCounterIconComponent.Kind())
	if ok {
		iconTransform, _ = ecs.Get(w, iconEntity, component.TransformComponent.Kind())
		iconSprite, _ = ecs.Get(w, iconEntity, component.SpriteComponent.Kind())
	}
	e, ok := ecs.First(w, component.TrophyCounterTextComponent.Kind())
	if ok {
		textEntity = e
		textTransform, _ = ecs.Get(w, textEntity, component.TransformComponent.Kind())
		textSprite, _ = ecs.Get(w, textEntity, component.SpriteComponent.Kind())
	}

	if counter == nil || tracker == nil || iconTransform == nil || iconSprite == nil || iconSprite.Image == nil || textTransform == nil || textSprite == nil {
		return
	}

	trackerChanged := false
	if tracker.Count < 0 {
		tracker.Count = 0
		trackerChanged = true
	}
	if trackerChanged {
		if trackerEntity != 0 {
			_ = ecs.Add(w, trackerEntity, component.TrophyTrackerComponent.Kind(), tracker)
		}
	}

	nextText := fmt.Sprintf("%d / 2", tracker.Count)
	textDirty := counter.RenderedText != nextText
	if counter.RenderedText != nextText {
		counter.RenderedText = nextText
		_ = ecs.Add(w, counterEntity, component.TrophyCounterComponent.Kind(), counter)
	}

	if textSprite.Image == nil || textDirty {
		textImage := ebiten.NewImage(trophyCounterTextW, trophyCounterTextH)
		ebitenutil.DebugPrintAt(textImage, nextText, 0, 0)
		textSprite.Image = textImage
		_ = ecs.Add(w, textEntity, component.SpriteComponent.Kind(), textSprite)
	}

	screenW, _ := ebiten.WindowSize()
	if screenW <= 0 {
		monitorW, _ := ebiten.Monitor().Size()
		screenW = monitorW
	}

	iconW := float64(iconSprite.Image.Bounds().Dx())
	iconH := float64(iconSprite.Image.Bounds().Dy())
	textW := float64(textSprite.Image.Bounds().Dx())
	textH := float64(textSprite.Image.Bounds().Dy())

	textX := float64(screenW) - trophyCounterPaddingX - textW
	iconX := textX - trophyCounterSpacing - iconW
	iconY := trophyCounterPaddingY
	textY := trophyCounterPaddingY + (iconH-textH)/2
	if textY < trophyCounterPaddingY {
		textY = trophyCounterPaddingY
	}

	iconTransform.X = iconX
	iconTransform.Y = iconY
	textTransform.X = textX
	textTransform.Y = textY
}
