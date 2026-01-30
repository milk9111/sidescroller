package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Prompt is a simple modal text input. When open it captures typed
// characters and calls the provided callback when the user presses Enter.
// Pressing Escape closes the prompt without invoking the callback.
type Prompt struct {
	open    bool
	label   string
	input   string
	onEnter func(string)
}

func NewPrompt() *Prompt { return &Prompt{} }

func (p *Prompt) IsOpen() bool { return p.open }

// Open shows the prompt with the given label, initial input, and callback.
func (p *Prompt) Open(label, initial string, onEnter func(string)) {
	p.label = label
	p.input = initial
	p.onEnter = onEnter
	p.open = true
}

// Close hides the prompt without invoking the callback.
func (p *Prompt) Close() {
	p.open = false
	p.label = ""
	p.input = ""
	p.onEnter = nil
}

// Update processes input for the prompt. Returns true if the prompt is open
// (so callers can early-return and avoid processing other input).
func (p *Prompt) Update() bool {
	if !p.open {
		return false
	}
	for _, r := range ebiten.InputChars() {
		if r == '\n' || r == '\r' {
			continue
		}
		p.input += string(r)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(p.input) > 0 {
			p.input = p.input[:len(p.input)-1]
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		// Close the prompt preemptively so callbacks can reopen it if they need a chained prompt.
		// Capture the current input to pass to the callback after closing.
		cur := p.input
		p.open = false
		if p.onEnter != nil {
			p.onEnter(cur)
		}
		// If the callback reopened the prompt, keep it open.
		if p.open {
			return true
		}
		// Otherwise fully close and clear state.
		p.Close()
		return false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		p.Close()
		return false
	}
	return true
}

// Draw renders the prompt overlay into the provided screen.
func (p *Prompt) Draw(screen *ebiten.Image) {
	if !p.open {
		return
	}
	sw := screen.Bounds().Dx()
	sh := screen.Bounds().Dy()
	o := &ebiten.DrawImageOptions{}
	back := ebiten.NewImage(sw, 48)
	back.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 0x88})
	o.GeoM.Translate(0, float64(sh/2-24))
	screen.DrawImage(back, o)
	prompt := p.label
	if prompt == "" {
		prompt = "Input:"
	}
	ebitenutil.DebugPrintAt(screen, prompt+" "+p.input, 16, sh/2-8)
}
