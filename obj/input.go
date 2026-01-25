package obj

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input holds current input state for movement and jumping.
type Input struct {
	// MoveX is -1 for left, 0 for none, +1 for right.
	MoveX float32
	// Jump is true while the jump key is pressed.
	Jump bool
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
	i.Jump = inpututil.IsKeyJustPressed(ebiten.KeySpace)
}
