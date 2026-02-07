package obj

import (
	"fmt"
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input holds current input state for movement and jumping.
type Input struct {
	// MoveX is -1 for left, 0 for none, +1 for right.
	MoveX float32
	// JumpPressed is true on the frame the jump key is pressed.
	JumpPressed bool
	// JumpHeld is true while the jump key is held down.
	JumpHeld bool
	// AimPressed is true on the frame the aim key (E) was pressed.
	AimPressed bool
	// AimHeld is true while the aim key is held.
	AimHeld bool
	// MouseLeftPressed is true on the frame the left mouse button was pressed.
	MouseLeftPressed bool
	// MouseWorldX/Y are the mouse cursor position in world coordinates (pixels).
	MouseWorldX float64
	MouseWorldY float64
	// DashPressed is true on the frame the dash key/button was pressed.
	DashPressed bool
	// LastAimAngle stores the last aiming angle (radians) used while in aim mode.
	LastAimAngle float64
	// LastAimValid indicates whether LastAimAngle contains a valid value.
	LastAimValid bool

	camera *Camera
	// previous trigger active states for edge detection on axis-mapped triggers
	prevLeftTriggerActive  bool
	prevRightTriggerActive bool
}

func NewInput(camera *Camera) *Input {
	return &Input{camera: camera}
}

// Update polls the keyboard and updates MoveX/Jump.
func (i *Input) Update() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		os.Exit(0)
	}

	mx, my := ebiten.CursorPosition()
	vx, vy := i.camera.ViewTopLeft()
	i.MouseWorldX = vx + float64(mx)/i.camera.Zoom()
	i.MouseWorldY = vy + float64(my)/i.camera.Zoom()

	var moveX float32
	// Keyboard D/A or arrows
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		moveX -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		moveX += 1
	}

	// Gamepad: if present, use left stick X axis as well and detect triggers
	ids := ebiten.GamepadIDs()
	var gpJumpJustPressed, gpJumpHeld, gpAimJustPressed, gpAimHeld, gpFireJustPressed bool
	var gpDashJustPressed bool
	if len(ids) > 0 {
		gid := ids[0]

		// Left stick X
		leftX := ebiten.StandardGamepadAxisValue(gid, ebiten.StandardGamepadAxisLeftStickHorizontal)
		if leftX < -0.3 {
			moveX = -1
		} else if leftX > 0.3 {
			moveX = 1
		}

		// Jump: map to standard primary button (use StandardGamepadButtonRightBottom as A/primary)
		gpJumpJustPressed = inpututil.IsStandardGamepadButtonJustPressed(gid, ebiten.StandardGamepadButtonRightBottom)
		gpJumpHeld = ebiten.IsStandardGamepadButtonPressed(gid, ebiten.StandardGamepadButtonRightBottom)

		// Triggers: use standard front-bottom buttons
		gpAimJustPressed = inpututil.IsStandardGamepadButtonJustPressed(gid, ebiten.StandardGamepadButtonFrontBottomLeft)
		gpAimHeld = ebiten.IsStandardGamepadButtonPressed(gid, ebiten.StandardGamepadButtonFrontBottomLeft)
		gpFireJustPressed = inpututil.IsStandardGamepadButtonJustPressed(gid, ebiten.StandardGamepadButtonFrontBottomRight)
		// X button (standard mapping: right-left)
		gpDashJustPressed = inpututil.IsStandardGamepadButtonJustPressed(gid, ebiten.StandardGamepadButtonRightLeft)

	}

	i.MoveX = moveX
	// Fire (attach) can come from mouse left or gamepad fire trigger/button
	i.MouseLeftPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || gpFireJustPressed

	// JumpPressed should be a true single-frame just-pressed signal to avoid
	// double-presses (which previously caused immediate double-jumps).
	i.JumpPressed = inpututil.IsKeyJustPressed(ebiten.KeySpace) || gpJumpJustPressed
	i.JumpHeld = ebiten.IsKeyPressed(ebiten.KeySpace) || gpJumpHeld

	// Aim toggle and hold: right mouse button or gamepad aim trigger/button
	i.AimPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) || gpAimJustPressed
	i.AimHeld = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) || gpAimHeld

	// Dash: Left Shift key or gamepad X button
	i.DashPressed = inpututil.IsKeyJustPressed(ebiten.KeyShiftLeft) || gpDashJustPressed

	// If a gamepad is connected, allow right stick to control the aim point
	// relative to the current mouse position.
	if len(ids) > 0 {
		gid := ids[0]
		// Prefer using the standard gamepad right-stick axes when available
		// but fall back to legacy axis index pairs for controllers that don't
		// expose the standard mapping.
		bestMag2 := 0.0
		var rx, ry float64
		// try standard right stick axes first
		sx := ebiten.StandardGamepadAxisValue(gid, ebiten.StandardGamepadAxisRightStickHorizontal)
		sy := ebiten.StandardGamepadAxisValue(gid, ebiten.StandardGamepadAxisRightStickVertical)
		if sx*sx+sy*sy > 0.0001 {
			rx = sx
			ry = sy
			bestMag2 = sx*sx + sy*sy
		} else {
			// fallback: scan likely axis indices to find the best right-stick pair.
			// Many controllers report sticks on axis pairs in the range 0..7.
			maxAxis := 8
			for a := 0; a < maxAxis; a++ {
				for b := 0; b < maxAxis; b++ {
					if a == b {
						continue
					}
					ax := ebiten.GamepadAxis(gid, a)
					ay := ebiten.GamepadAxis(gid, b)
					m2 := ax*ax + ay*ay
					if m2 > bestMag2 {
						bestMag2 = m2
						rx = ax
						ry = ay
					}
				}
			}
			if bestMag2 < 0.0001 {
				// no stick movement detected; helpful debug when users report
				// the stick stops working. Prints controller id and sampled axes.
				vals := ""
				for a := 0; a < maxAxis; a++ {
					vals += fmt.Sprintf("%0.3f,", ebiten.GamepadAxis(gid, a))
				}
				fmt.Printf("[input] right-stick fallback found no movement (bestMag2=%0.6f) axes=%s\n", bestMag2, vals)
			}
		}

		if bestMag2 > 0.01 {
			mag := math.Hypot(rx, ry)
			if mag > 0 {
				nx := rx / mag
				ny := ry / mag
				i.MouseWorldX += nx * 200
				i.MouseWorldY += ny * 200
				i.LastAimAngle = math.Atan2(ny, nx)
				i.LastAimValid = true
			}
		}
	}
}
