package obj

import (
	"math"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/common"
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

	camera *Camera
	// reference to the player (set by the game) so input can adjust
	// aim coordinates while in aim mode (right stick).
	Player *Player
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

	// If the player is currently in aim mode and a gamepad is connected,
	// use the right stick to control the aim point instead of the OS cursor.
	if i.Player != nil && i.Player.IsAiming() && len(ids) > 0 {
		gid := ids[0]
		// Recompute right-stick pair here (keeps logic simple and robust).
		candidatePairs := [][2]int{{3, 4}}
		bestMag2 := 0.0
		var rx, ry float64
		for _, p := range candidatePairs {
			ax := ebiten.GamepadAxis(gid, p[0])
			ay := ebiten.GamepadAxis(gid, p[1])
			m2 := ax*ax + ay*ay
			if m2 > bestMag2 {
				bestMag2 = m2
				rx = ax
				ry = ay
			}
		}

		if bestMag2 > 0.01 {
			// aim radius: use a very large distance so the aim ray is effectively
			// infinite. Prefer using level bounds when available so the value is
			// reasonable for the current level.
			aimRadius := 100000.0
			if i.Player != nil && i.Player.CollisionWorld != nil && i.Player.CollisionWorld.level != nil {
				lw := float64(i.Player.CollisionWorld.level.Width * common.TileSize)
				lh := float64(i.Player.CollisionWorld.level.Height * common.TileSize)
				maxDim := math.Max(lw, lh)
				aimRadius = maxDim * 2.0
			}
			cx := float64(i.Player.X + float32(i.Player.Width)/2.0)
			cy := float64(i.Player.Y + float32(i.Player.Height)/2.0)

			// Do not invert Y here; the axis values are already in the
			// controller's coordinate space and should map directly.
			// Normalize direction and place reticle far away (effectively infinite)
			mag := math.Hypot(rx, ry)
			if mag > 0 {
				nx := rx / mag
				ny := ry / mag
				i.MouseWorldX = cx + nx*aimRadius
				i.MouseWorldY = cy + ny*aimRadius
			}
		}
	}
}
