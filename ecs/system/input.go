package system

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type InputSystem struct{}

func NewInputSystem() *InputSystem {
	return &InputSystem{}
}

func (i *InputSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	const stickDeadzone = 0.2

	left := ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	jump := ebiten.IsKeyPressed(ebiten.KeySpace)
	jumpPressed := inpututil.IsKeyJustPressed(ebiten.KeySpace)
	aim := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	aimX := 0.0
	aimY := 0.0

	moveX := 0.0
	if left {
		moveX -= 1
	}
	if right {
		moveX += 1
	}

	if gamepads := ebiten.GamepadIDs(); len(gamepads) > 0 {
		id := gamepads[0]
		leftX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		if math.Abs(leftX) > stickDeadzone {
			moveX = leftX
		}

		jump = jump || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom)
		jumpPressed = jumpPressed || inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom)

		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonFrontBottomLeft) {
			aim = true
		}

		rx := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickHorizontal)
		ry := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickVertical)
		if math.Hypot(rx, ry) > stickDeadzone {
			aimX = rx
			aimY = ry
		}
	}

	for _, e := range w.Query(component.InputComponent.Kind()) {
		input, ok := ecs.Get(w, e, component.InputComponent)
		if !ok {
			input = component.Input{}
		}
		input.MoveX = moveX
		input.Jump = jump
		input.JumpPressed = jumpPressed
		input.Aim = aim
		input.AimX = aimX
		input.AimY = aimY
		if err := ecs.Add(w, e, component.InputComponent, input); err != nil {
			panic("input system: update input: " + err.Error())
		}
	}
}
