package system

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type DialogueInputSystem struct{}

func NewDialogueInputSystem() *DialogueInputSystem {
	return &DialogueInputSystem{}
}

func (s *DialogueInputSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	pressed := inpututil.IsKeyJustPressed(ebiten.KeyZ)
	usingGamepad := false

	if gamepads := ebiten.GamepadIDs(); len(gamepads) > 0 {
		usingGamepad = true
		id := gamepads[0]
		pressed = pressed || inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightLeft)
	}

	ecs.ForEach(w, component.DialogueInputComponent.Kind(), func(e ecs.Entity, input *component.DialogueInput) {
		if input == nil {
			return
		}
		input.Pressed = pressed
		input.UsingGamepad = usingGamepad
	})
}
