package system

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const transitionUpAxisThreshold = -0.5

type TransitionInputSystem struct {
	stickUpLastFrame bool
}

func NewTransitionInputSystem() *TransitionInputSystem {
	return &TransitionInputSystem{}
}

func (s *TransitionInputSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	pressed := inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp)
	usingGamepad := false
	stickUp := false

	if gamepads := ebiten.GamepadIDs(); len(gamepads) > 0 {
		usingGamepad = true
		id := gamepads[0]
		pressed = pressed || inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftTop)
		stickUp = ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical) <= transitionUpAxisThreshold
		if stickUp && !s.stickUpLastFrame {
			pressed = true
		}
	}

	s.stickUpLastFrame = stickUp

	ecs.ForEach(w, component.TransitionInputComponent.Kind(), func(_ ecs.Entity, input *component.TransitionInput) {
		if input == nil {
			return
		}
		input.UpPressed = pressed
		input.UsingGamepad = usingGamepad
	})
}
