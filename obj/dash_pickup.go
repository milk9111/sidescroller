package obj

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
)

// DashPickup represents a placed dash pickup that floats and can be triggered by the player.
type DashPickup struct {
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

// NewDashPickup creates a DashPickup at world pixel (x,y). spritePath may be empty.
func NewDashPickup(x, y float32, spritePath string, onPickup func()) *DashPickup {
	di := &DashPickup{
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
			di.img = im
		}
	}

	if di.img == nil {
		// make small placeholder (magenta square)
		p := ebiten.NewImage(common.TileSize, common.TileSize)
		p.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff})
		di.placeholder = p
	}

	return di
}

// Update checks collision with the player and triggers on enter.
func (d *DashPickup) Update(p *Player) {
	if d == nil || p == nil || d.Disabled {
		return
	}

	t := float64(time.Now().UnixNano()) / 1e9
	yOffset := float32(math.Sin(t*d.frequency+d.phase) * d.amplitude)

	entL := d.X
	entT := d.Y + yOffset
	entR := d.X + float32(common.TileSize)
	entB := entT + float32(common.TileSize)

	pL := p.X
	pT := p.Y
	pR := p.X + float32(p.Width)
	pB := p.Y + float32(p.Height)

	colliding := !(pR < entL || pL > entR || pB < entT || pT > entB)
	if colliding && !d.touched {
		d.onPickup()
		d.touched = true
	}

	if !colliding {
		d.touched = false
	}
}

// Draw draws the dash pickup into the provided world image (world is already camera-transformed space).
// camX/camY/zoom are supplied by the caller (camera.Render passes view top-left and zoom).
func (d *DashPickup) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if d == nil || d.Disabled {
		return
	}

	t := float64(time.Now().UnixNano()) / 1e9
	yOffset := float32(math.Sin(t*d.frequency+d.phase) * d.amplitude)
	img := d.img
	if img == nil {
		img = d.placeholder
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
	tx := (float64(d.X) - camX)
	ty := (float64(d.Y+yOffset) - camY)
	op.GeoM.Translate(tx*zoom, ty*zoom)
	screen.DrawImage(img, op)
}
