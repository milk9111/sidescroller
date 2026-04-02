package entity

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	DebugMessageDefaultWidth  = 420
	DebugMessageDefaultHeight = 96
	DebugMessageDefaultFrames = 30 * 60
	DebugMessageRenderLayer   = 1100
	DebugMessageTopY          = 44.0
	debugMessagePaddingX      = 8
	debugMessagePaddingY      = 4
	debugMessageMinX          = 8.0
	debugMessagePersistentID  = "debug_message"
)

func NewDebugMessage(w *ecs.World) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("debug message: world is nil")
	}

	ent := ecs.CreateEntity(w)
	if err := ecs.Add(w, ent, component.PersistentComponent.Kind(), &component.Persistent{ID: debugMessagePersistentID, KeepOnLevelChange: true, KeepOnReload: false}); err != nil {
		return 0, fmt.Errorf("debug message: add persistent: %w", err)
	}
	if err := ecs.Add(w, ent, component.DebugMessageComponent.Kind(), &component.DebugMessage{Width: DebugMessageDefaultWidth, Height: DebugMessageDefaultHeight}); err != nil {
		return 0, fmt.Errorf("debug message: add state: %w", err)
	}
	if err := ecs.Add(w, ent, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{}); err != nil {
		return 0, fmt.Errorf("debug message: add screen space: %w", err)
	}
	if err := ecs.Add(w, ent, component.TransformComponent.Kind(), &component.Transform{X: debugMessageX(DebugMessageDefaultWidth), Y: DebugMessageTopY, ScaleX: 1, ScaleY: 1}); err != nil {
		return 0, fmt.Errorf("debug message: add transform: %w", err)
	}
	if err := ecs.Add(w, ent, component.SpriteComponent.Kind(), &component.Sprite{Disabled: true}); err != nil {
		return 0, fmt.Errorf("debug message: add sprite: %w", err)
	}
	if err := ecs.Add(w, ent, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: DebugMessageRenderLayer}); err != nil {
		return 0, fmt.Errorf("debug message: add render layer: %w", err)
	}

	return ent, nil
}

func ShowDebugMessage(w *ecs.World, width, height int, message string) error {
	return ShowTimedDebugMessage(w, width, height, message, DebugMessageDefaultFrames)
}

func ShowTimedDebugMessage(w *ecs.World, width, height int, message string, frames int) error {
	if w == nil {
		return fmt.Errorf("debug message: world is nil")
	}
	if width <= 0 {
		return fmt.Errorf("debug message: width must be positive")
	}
	if height <= 0 {
		return fmt.Errorf("debug message: height must be positive")
	}

	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("debug message: message cannot be empty")
	}

	ent, err := ensureDebugMessageEntity(w)
	if err != nil {
		return err
	}

	debugMessage, _ := ecs.Get(w, ent, component.DebugMessageComponent.Kind())
	if debugMessage == nil {
		debugMessage = &component.DebugMessage{}
		if err := ecs.Add(w, ent, component.DebugMessageComponent.Kind(), debugMessage); err != nil {
			return fmt.Errorf("debug message: add state: %w", err)
		}
	}

	transform, _ := ecs.Get(w, ent, component.TransformComponent.Kind())
	if transform == nil {
		transform = &component.Transform{}
		if err := ecs.Add(w, ent, component.TransformComponent.Kind(), transform); err != nil {
			return fmt.Errorf("debug message: add transform: %w", err)
		}
	}

	sprite, _ := ecs.Get(w, ent, component.SpriteComponent.Kind())
	if sprite == nil {
		sprite = &component.Sprite{}
		if err := ecs.Add(w, ent, component.SpriteComponent.Kind(), sprite); err != nil {
			return fmt.Errorf("debug message: add sprite: %w", err)
		}
	}

	debugMessage.Width = width
	debugMessage.Height = height
	debugMessage.Message = message
	if frames > 0 {
		debugMessage.RemainingFrames = frames
	} else {
		debugMessage.RemainingFrames = -1
	}

	transform.X = debugMessageX(width)
	transform.Y = DebugMessageTopY
	if transform.ScaleX == 0 {
		transform.ScaleX = 1
	}
	if transform.ScaleY == 0 {
		transform.ScaleY = 1
	}

	sprite.Image = renderDebugMessageImage(width, height, message)
	sprite.Disabled = false

	_ = ecs.Add(w, ent, component.ScreenSpaceComponent.Kind(), &component.ScreenSpace{})
	_ = ecs.Add(w, ent, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: DebugMessageRenderLayer})

	return nil
}

func HideDebugMessage(w *ecs.World) error {
	if w == nil {
		return fmt.Errorf("debug message: world is nil")
	}

	ent, ok := ecs.First(w, component.DebugMessageComponent.Kind())
	if !ok {
		return nil
	}

	if debugMessage, ok := ecs.Get(w, ent, component.DebugMessageComponent.Kind()); ok && debugMessage != nil {
		debugMessage.Message = ""
		debugMessage.RemainingFrames = 0
	}

	if sprite, ok := ecs.Get(w, ent, component.SpriteComponent.Kind()); ok && sprite != nil {
		sprite.Disabled = true
		sprite.Image = nil
	}

	return nil
}

func ensureDebugMessageEntity(w *ecs.World) (ecs.Entity, error) {
	if ent, ok := ecs.First(w, component.DebugMessageComponent.Kind()); ok {
		return ent, nil
	}
	return NewDebugMessage(w)
}

func renderDebugMessageImage(width, height int, message string) *ebiten.Image {
	img := ebiten.NewImage(width, height)
	img.Fill(color.NRGBA{A: 170})
	ebitenutil.DebugPrintAt(img, message, debugMessagePaddingX, debugMessagePaddingY)
	return img
}

func debugMessageX(width int) float64 {
	x := float64(common.BaseWidth-width) / 2
	if x < debugMessageMinX {
		return debugMessageMinX
	}
	return x
}
