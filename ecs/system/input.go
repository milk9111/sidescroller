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
	// look input for camera (W/Up = look up, S/Down = look down)
	lookUp := ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	lookDown := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	lookY := 0.0
	if lookUp {
		lookY -= 1
	}
	if lookDown {
		lookY += 1
	}
	jumpPressed := inpututil.IsKeyJustPressed(ebiten.KeySpace)
	aim := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	anchorPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && aim
	attackPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !aim
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
		attackPressed = attackPressed || inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightLeft)

		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonFrontBottomLeft) {
			aim = true
		}

		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonFrontBottomRight) {
			anchorPressed = true
		}

		rx := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickHorizontal)
		ry := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickVertical)
		if math.Hypot(rx, ry) > stickDeadzone {
			aimX = rx
			aimY = ry
		}

		// allow left stick vertical to control look if it's being used
		ly := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if math.Abs(ly) > stickDeadzone {
			// Gamepad axis vertical is typically -1 = up, +1 = down
			lookY = float64(ly)
		}
	}

	ecs.ForEach(w, component.InputComponent.Kind(), func(e ecs.Entity, input *component.Input) {
		input.MoveX = moveX
		input.Jump = jump
		input.JumpPressed = jumpPressed
		input.Aim = aim
		input.AimX = aimX
		input.AimY = aimY
		input.LookY = lookY
		input.AnchorPressed = anchorPressed
		input.AttackPressed = attackPressed
		// if err := ecs.Add(w, e, component.InputComponent.Kind(), input); err != nil {
		// 	panic("input system: update input: " + err.Error())
		// }
	})
}
