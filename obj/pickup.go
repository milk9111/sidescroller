//go:build legacy
// +build legacy

package obj

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
)

// Pickup represents a placed pickup that floats and can be triggered by the player.
type Pickup struct {
	X, Y     float32 // world pixel top-left
	Disabled bool

	spritePath  string
	img         *ebiten.Image
	phase       float64
	amplitude   float64
	frequency   float64
	touched     bool
	placeholder *ebiten.Image

	onPickup func()
}

// NewPickup creates a Pickup at world pixel (x,y). spritePath may be empty.
func NewPickup(x, y float32, spritePath string, onPickup func()) *Pickup {
	p := &Pickup{
		X:          x,
		Y:          y,
		Disabled:   false,
		spritePath: spritePath,
		amplitude:  4.0,
		frequency:  2.0,
		phase:      float64(int(x)%7) * 0.3,
		onPickup:   onPickup,
	}

	if spritePath != "" {
		if im, err := assets.LoadImage(spritePath); err == nil {
			p.img = im
		}
	}

	if p.img == nil {
		// make small placeholder (magenta square)
		img := ebiten.NewImage(common.TileSize, common.TileSize)
		img.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff})
		p.placeholder = img
	}

	return p
}

// Update checks collision with the player and triggers on enter.
func (p *Pickup) Update(player *Player) {
	if p == nil || player == nil || p.Disabled {
		return
	}

	t := float64(time.Now().UnixNano()) / 1e9
	yOffset := float32(math.Sin(t*p.frequency+p.phase) * p.amplitude)

	entL := p.X
	entT := p.Y + yOffset
	entR := p.X + float32(common.TileSize)
	entB := entT + float32(common.TileSize)

	pL := player.X
	pT := player.Y
	pR := player.X + float32(player.Width)
	pB := player.Y + float32(player.Height)

	colliding := !(pR < entL || pL > entR || pB < entT || pT > entB)
	if colliding && !p.touched {
		if p.onPickup != nil {
			p.onPickup()
		}
		p.touched = true
	}

	if !colliding {
		p.touched = false
	}
}

// Draw draws the pickup into the provided world image (world is already camera-transformed space).
// camX/camY/zoom are supplied by the caller (camera.Render passes view top-left and zoom).
func (p *Pickup) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if p == nil || p.Disabled {
		return
	}

	t := float64(time.Now().UnixNano()) / 1e9
	yOffset := float32(math.Sin(t*p.frequency+p.phase) * p.amplitude)
	img := p.img
	if img == nil {
		img = p.placeholder
	}

	if img == nil {
		return
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return
	}

	op := &ebiten.DrawImageOptions{}
	// scale to tile size
	op.GeoM.Scale(float64(common.TileSize)/float64(w)*zoom, float64(common.TileSize)/float64(h)*zoom)
	// translate into camera-local coordinates and apply yOffset
	tx := (float64(p.X) - camX)
	ty := (float64(p.Y+yOffset) - camY)
	op.GeoM.Translate(tx*zoom, ty*zoom)
	screen.DrawImage(img, op)
}
