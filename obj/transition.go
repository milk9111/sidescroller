package obj

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// Transition manages a fade-to-black, level load, fade-from-black sequence.
type Transition struct {
	Active    bool
	Phase     int // 1: fade-in, 2: loading, 3: fade-out
	Frames    int
	Duration  int
	Target    string
	LinkID    string
	Direction string
	overlay   *ebiten.Image
	// OnStart is called when the fade-in completes and the level should be loaded.
	// Signature: func(target, linkID, direction string)
	OnStart func(target, linkID, direction string)
}

func NewTransition() *Transition {
	return &Transition{
		Active:   false,
		Phase:    0,
		Frames:   0,
		Duration: 20,
		overlay:  func() *ebiten.Image { img := ebiten.NewImage(1, 1); img.Fill(color.Black); return img }(),
	}
}

// Enter starts a transition to the provided target/link/direction.
func (t *Transition) Enter(target, linkID, direction string) {
	if t.Active {
		return
	}
	t.Active = true
	t.Phase = 1
	t.Frames = 0
	t.Target = target
	t.LinkID = linkID
	t.Direction = direction
}

// Update advances the transition. It will invoke `OnStart` at the midpoint
// (after fade-in). If the transition is active, Update returns true to
// indicate the caller should skip normal world updates.
func (t *Transition) Update() bool {
	if !t.Active {
		return false
	}
	t.Frames++
	switch t.Phase {
	case 1: // fade-in
		if t.Frames >= t.Duration {
			t.Phase = 2
			t.Frames = 0
			// invoke load callback (if provided) so caller can perform the
			// Game-specific level load and setup. This replaces direct Game
			// mutations inside the obj package.
			if t.OnStart != nil {
				t.OnStart(t.Target, t.LinkID, t.Direction)
			}

			// move to fade-out
			t.Phase = 3
			t.Frames = 0
		}
	case 3: // fade-out
		if t.Frames >= t.Duration {
			t.Active = false
			t.Phase = 0
			t.Frames = 0
			t.Target = ""
			t.LinkID = ""
			t.Direction = ""
		}
	}
	return true
}

// Draw draws the fade overlay onto the provided screen.
func (t *Transition) Draw(screen *ebiten.Image) {
	if !t.Active {
		return
	}
	var alpha float64
	switch t.Phase {
	case 1:
		alpha = float64(t.Frames) / float64(t.Duration)
		if alpha > 1 {
			alpha = 1
		}
	case 2:
		alpha = 1
	case 3:
		alpha = 1 - float64(t.Frames)/float64(t.Duration)
		if alpha < 0 {
			alpha = 0
		}
	}

	if alpha <= 0 {
		return
	}

	w, h := screen.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	var cm ebiten.ColorM
	cm.Scale(1, 1, 1, alpha)
	op.ColorM = cm
	screen.DrawImage(t.overlay, op)
}
