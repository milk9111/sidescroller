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
}

func NewInput() *Input { return &Input{} }

// Update polls the keyboard and updates MoveX/Jump.
func (i *Input) Update() {
	var mx float32
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		mx -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		mx += 1
	}
	i.MoveX = mx
	i.JumpPressed = inpututil.IsKeyJustPressed(ebiten.KeySpace)
	i.JumpHeld = ebiten.IsKeyPressed(ebiten.KeySpace)
	i.AimPressed = inpututil.IsKeyJustPressed(ebiten.KeyE)
	i.AimHeld = ebiten.IsKeyPressed(ebiten.KeyE)
	// Mouse world position and MouseLeftPressed are set by the caller (Game)
}
