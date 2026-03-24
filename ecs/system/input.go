package system

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const autoAnchorDoubleClickWindowFrames = 20
const gamepadUpwardAttackThreshold = -0.6

type InputSystem struct {
	frame                    int
	lastRightMousePressFrame int
}

func NewInputSystem() *InputSystem {
	return &InputSystem{lastRightMousePressFrame: -1}
}

func registerDoublePress(frame, lastPressFrame, window int) (bool, int) {
	if lastPressFrame >= 0 && frame-lastPressFrame <= window {
		return true, -1
	}
	return false, frame
}

func shouldTriggerUpwardAttack(attackPressed bool, aimY float64, keyboardUpPressed bool, usingGamepad bool) bool {
	if !attackPressed {
		return false
	}
	if keyboardUpPressed {
		return true
	}
	if usingGamepad {
		return aimY <= gamepadUpwardAttackThreshold
	}
	return aimY < 0
}

func (i *InputSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	i.frame++

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		if _, ok := ecs.First(w, component.ResetToInitialLevelRequestComponent.Kind()); !ok {
			ent := ecs.CreateEntity(w)
			_ = ecs.Add(w, ent, component.ResetToInitialLevelRequestComponent.Kind(), &component.ResetToInitialLevelRequest{})
		}
	}

	const stickDeadzone = 0.2

	left := ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	jump := ebiten.IsKeyPressed(ebiten.KeySpace)
	keyboardUpPressed := ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp)
	// Keyboard look only supports looking down so W/Up can remain dedicated to upward attacks.
	lookDown := ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown)
	lookY := 0.0
	if lookDown {
		lookY += 1
	}
	jumpPressed := inpututil.IsKeyJustPressed(ebiten.KeySpace)
	aim := ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	anchorPressed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && aim
	autoAnchorPressed := inpututil.IsKeyJustPressed(ebiten.KeyControlLeft) || inpututil.IsKeyJustPressed(ebiten.KeyControlRight)
	anchorReelIn := ebiten.IsKeyPressed(ebiten.KeyQ)
	anchorReelOut := ebiten.IsKeyPressed(ebiten.KeyE)
	attackPressed := (inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || inpututil.IsKeyJustPressed(ebiten.KeyZ)) && !aim
	aimX := 0.0
	aimY := 0.0

	moveX := 0.0
	if left {
		moveX -= 1
	}
	if right {
		moveX += 1
	}

	var usingGamepad bool
	var anchorReleasePressed bool
	if gamepads := ebiten.GamepadIDs(); len(gamepads) > 0 {
		usingGamepad = true
		id := gamepads[0]

		anchorExists := false
		if _, ok := ecs.First(w, component.AnchorTagComponent.Kind()); ok {
			anchorExists = true
		}

		leftX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		if math.Abs(leftX) > stickDeadzone {
			moveX = leftX
		}

		jump = jump || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom)
		jumpPressed = jumpPressed || inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom)
		if anchorExists && ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightLeft) {
			anchorReelIn = true
		} else if anchorExists && ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightTop) {
			anchorReelOut = true
		} else {
			attackPressed = attackPressed || inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightLeft)
		}

		if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonFrontBottomLeft) {
			aim = true
		}

		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonFrontBottomRight) {
			if _, ok := ecs.First(w, component.AnchorTagComponent.Kind()); ok { // if an anchor exists, this button releases it instead of firing a new one
				anchorReleasePressed = true
			} else if aim {
				anchorPressed = true
			} else {
				autoAnchorPressed = true
			}
		}

		lx := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		ly := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if math.Hypot(lx, ly) > stickDeadzone {
			aimX = lx
			aimY = ly
		}

		// allow left stick vertical to control look if it's being used
		ry := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisRightStickVertical)
		if math.Abs(ry) > stickDeadzone {
			// Gamepad axis vertical is typically -1 = up, +1 = down
			lookY = float64(ry)
		}
	}

	upwardAttackPressed := false
	if shouldTriggerUpwardAttack(attackPressed, aimY, keyboardUpPressed, usingGamepad) {
		upwardAttackPressed = true
		attackPressed = false
	}

	ecs.ForEach(w, component.InputComponent.Kind(), func(e ecs.Entity, input *component.Input) {
		if input.Disabled {
			input.MoveX = 0
			input.Jump = false
			input.JumpPressed = false
			input.Aim = false
			input.AimX = 0
			input.AimY = 0
			input.LookY = 0
			input.AnchorPressed = false
			input.AutoAnchorPressed = false
			input.AnchorReelIn = false
			input.AnchorReelOut = false
			input.AttackPressed = false
			input.UpwardAttackPressed = false
			input.AnchorReleasePressed = false
			input.UsingGamepad = false
			return
		}

		input.MoveX = moveX
		input.Jump = jump
		input.JumpPressed = jumpPressed
		input.Aim = aim
		input.AimX = aimX
		input.AimY = aimY
		input.LookY = lookY
		input.AnchorPressed = anchorPressed
		input.AutoAnchorPressed = autoAnchorPressed
		input.AnchorReelIn = anchorReelIn
		input.AnchorReelOut = anchorReelOut
		input.AttackPressed = attackPressed
		input.UpwardAttackPressed = upwardAttackPressed
		input.AnchorReleasePressed = anchorReleasePressed
		input.UsingGamepad = usingGamepad
	})
}
