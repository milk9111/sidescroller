package obj

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Camera renders the world centered on a given world coordinate and supports zoom.
type Camera struct {
	PosX float64
	PosY float64

	screenW int
	screenH int
	zoom    float64
	off     *ebiten.Image

	// smoothing factor (0..1). higher -> faster follow. e.g. 0.15
	smooth float64
	// world bounds in pixels (0 means unbounded)
	worldW float64
	worldH float64
}

// NewCamera creates a camera with the given logical screen size and initial zoom.
func NewCamera(screenW, screenH int, zoom float64) *Camera {
	c := &Camera{screenW: screenW, screenH: screenH, zoom: zoom, smooth: 0.15}
	c.off = ebiten.NewImage(screenW, screenH)
	// default position at screen center in world coords
	c.PosX = float64(screenW) / 2.0
	c.PosY = float64(screenH) / 2.0
	return c
}

// SetZoom updates the camera zoom.
func (c *Camera) SetZoom(z float64) {
	if z <= 0 {
		return
	}
	c.zoom = z
}

// SetScreenSize updates the logical screen size used by the camera.
func (c *Camera) SetScreenSize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	if c.screenW == w && c.screenH == h {
		return
	}
	c.screenW = w
	c.screenH = h
	c.off = nil
}

// SetWorldBounds sets the world pixel dimensions for clamping camera position.
func (c *Camera) SetWorldBounds(w, h int) {
	c.worldW = float64(w)
	c.worldH = float64(h)
}

func (c *Camera) SetSmooth(f float64) {
	if f < 0 {
		f = 0
	}
	c.smooth = f
}

// ViewTopLeft returns the world-space top-left of the current view.
func (c *Camera) ViewTopLeft() (float64, float64) {
	if c.zoom == 0 {
		return c.PosX, c.PosY
	}
	viewW := float64(c.screenW) / c.zoom
	viewH := float64(c.screenH) / c.zoom
	return c.PosX - viewW/2.0, c.PosY - viewH/2.0
}

// Zoom returns the current camera zoom.
func (c *Camera) Zoom() float64 {
	return c.zoom
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Render draws the world by first invoking drawWorld with the offscreen image
// (which should be treated as world-space with the same logical size as the
// game), then draws the offscreen image onto the provided screen such that
// the world point (centerX, centerY) is mapped to the center of the screen.
// Update moves the camera toward the target world coordinate. Call from the
// fixed-rate Update loop to get consistent smoothing.
func (c *Camera) Update(targetX, targetY float64) {
	// Smoothly follow the requested center point using linear interpolation
	if c.smooth <= 0 {
		c.PosX = targetX
		c.PosY = targetY
	} else {
		c.PosX += (targetX - c.PosX) * c.smooth
		c.PosY += (targetY - c.PosY) * c.smooth
	}

	// snap position to 1/zoom grid to align source texels to integer screen pixels
	if c.zoom != 0 {
		c.PosX = math.Round(c.PosX*c.zoom) / c.zoom
		c.PosY = math.Round(c.PosY*c.zoom) / c.zoom
	}

	// clamp to world bounds if provided
	// compute half view size in world coordinates
	viewW := float64(c.screenW) / c.zoom
	viewH := float64(c.screenH) / c.zoom
	halfW := viewW / 2.0
	halfH := viewH / 2.0
	if c.worldW > 0 {
		minX := halfW
		maxX := c.worldW - halfW
		if maxX < minX {
			// world smaller than view: center on world
			c.PosX = c.worldW / 2.0
		} else {
			c.PosX = clamp(c.PosX, minX, maxX)
		}
	}

	if c.worldH > 0 {
		minY := halfH
		maxY := c.worldH - halfH
		if maxY < minY {
			c.PosY = c.worldH / 2.0
		} else {
			c.PosY = clamp(c.PosY, minY, maxY)
		}
	}
}

// SnapTo immediately sets the camera center to the given world coordinates
// and applies rounding/clamping as performed by Update. Use this when an
// immediate, non-smoothed camera placement is required (e.g. after a level
// load) to ensure the view is constrained to world bounds.
func (c *Camera) SnapTo(x, y float64) {
	c.PosX = x
	c.PosY = y

	// snap position to 1/zoom grid to align source texels to integer screen pixels
	if c.zoom != 0 {
		c.PosX = math.Round(c.PosX*c.zoom) / c.zoom
		c.PosY = math.Round(c.PosY*c.zoom) / c.zoom
	}

	// clamp to world bounds if provided
	viewW := float64(c.screenW) / c.zoom
	viewH := float64(c.screenH) / c.zoom
	halfW := viewW / 2.0
	halfH := viewH / 2.0
	if c.worldW > 0 {
		minX := halfW
		maxX := c.worldW - halfW
		if maxX < minX {
			c.PosX = c.worldW / 2.0
		} else {
			c.PosX = clamp(c.PosX, minX, maxX)
		}
	}

	if c.worldH > 0 {
		minY := halfH
		maxY := c.worldH - halfH
		if maxY < minY {
			c.PosY = c.worldH / 2.0
		} else {
			c.PosY = clamp(c.PosY, minY, maxY)
		}
	}
}

// Render draws the world by first invoking drawWorld with the offscreen image
// (which should be treated as view-space sized to the screen), then draws the
// offscreen image onto the provided screen. The caller should draw with
// camX/camY offsets based on ViewTopLeft().
func (c *Camera) Render(screen *ebiten.Image, drawWorld func(world *ebiten.Image)) {
	if c.off == nil {
		c.off = ebiten.NewImage(c.screenW, c.screenH)
	}

	// clear offscreen and let caller draw the world into it
	c.off.Clear()
	if drawWorld != nil {
		drawWorld(c.off)
	}

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(c.off, op)
}
