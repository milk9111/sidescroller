package obj

import (
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

	camera *Camera
}

func NewInput(camera *Camera) *Input {
	return &Input{camera: camera}
}

// Update polls the keyboard and updates MoveX/Jump.
func (i *Input) Update() {
	mx, my := ebiten.CursorPosition()
	vx, vy := i.camera.ViewTopLeft()
	i.MouseWorldX = vx + float64(mx)/i.camera.Zoom()
	i.MouseWorldY = vy + float64(my)/i.camera.Zoom()

	var moveX float32
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		moveX -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		moveX += 1
	}
	i.MoveX = moveX
	i.MouseLeftPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	i.JumpPressed = inpututil.IsKeyJustPressed(ebiten.KeySpace)
	i.JumpHeld = ebiten.IsKeyPressed(ebiten.KeySpace)
	i.AimPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	i.AimHeld = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
}
