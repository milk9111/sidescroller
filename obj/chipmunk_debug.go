package obj

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/jakecoffman/cp"
)

// DebugDraw renders chipmunk shapes for debugging.
func (cw *CollisionWorld) DebugDraw(screen *ebiten.Image) {
	if cw == nil || cw.space == nil || screen == nil {
		return
	}
	cp.DrawSpace(cw.space, &chipmunkDrawer{screen: screen})
}

type chipmunkDrawer struct {
	screen *ebiten.Image
}

func (d *chipmunkDrawer) DrawCircle(pos cp.Vector, angle, radius float64, outline, fill cp.FColor, data interface{}) {
	if d.screen == nil {
		return
	}
	c := fcolorToRGBA(outline)
	steps := 20
	prev := cp.Vector{X: pos.X + radius, Y: pos.Y}
	for i := 1; i <= steps; i++ {
		th := float64(i) * (2 * math.Pi / float64(steps))
		cur := cp.Vector{X: pos.X + math.Cos(th)*radius, Y: pos.Y + math.Sin(th)*radius}
		ebitenutil.DrawLine(d.screen, prev.X, prev.Y, cur.X, cur.Y, c)
		prev = cur
	}
	// draw angle indicator
	ax := pos.X + math.Cos(angle)*radius
	ay := pos.Y + math.Sin(angle)*radius
	ebitenutil.DrawLine(d.screen, pos.X, pos.Y, ax, ay, c)
}

func (d *chipmunkDrawer) DrawSegment(a, b cp.Vector, fill cp.FColor, data interface{}) {
	if d.screen == nil {
		return
	}
	ebitenutil.DrawLine(d.screen, a.X, a.Y, b.X, b.Y, fcolorToRGBA(fill))
}

func (d *chipmunkDrawer) DrawFatSegment(a, b cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
	if d.screen == nil {
		return
	}
	ebitenutil.DrawLine(d.screen, a.X, a.Y, b.X, b.Y, fcolorToRGBA(outline))
	if radius > 0 {
		d.DrawCircle(a, 0, radius, outline, fill, data)
		d.DrawCircle(b, 0, radius, outline, fill, data)
	}
}

func (d *chipmunkDrawer) DrawPolygon(count int, verts []cp.Vector, radius float64, outline, fill cp.FColor, data interface{}) {
	if d.screen == nil || count == 0 {
		return
	}
	c := fcolorToRGBA(outline)
	for i := 0; i < count; i++ {
		j := (i + 1) % count
		a := verts[i]
		b := verts[j]
		ebitenutil.DrawLine(d.screen, a.X, a.Y, b.X, b.Y, c)
	}
	if radius > 0 {
		for i := 0; i < count; i++ {
			d.DrawCircle(verts[i], 0, radius, outline, fill, data)
		}
	}
}

func (d *chipmunkDrawer) DrawDot(size float64, pos cp.Vector, fill cp.FColor, data interface{}) {
	if d.screen == nil {
		return
	}
	c := fcolorToRGBA(fill)
	l := size / 2
	ebitenutil.DrawLine(d.screen, pos.X-l, pos.Y, pos.X+l, pos.Y, c)
	ebitenutil.DrawLine(d.screen, pos.X, pos.Y-l, pos.X, pos.Y+l, c)
}

func (d *chipmunkDrawer) Flags() uint {
	return cp.DRAW_SHAPES
}

func (d *chipmunkDrawer) OutlineColor() cp.FColor {
	return cp.FColor{R: 0.2, G: 1.0, B: 0.2, A: 1.0}
}

func (d *chipmunkDrawer) ShapeColor(shape *cp.Shape, data interface{}) cp.FColor {
	if shape == nil {
		return cp.FColor{R: 1, G: 1, B: 1, A: 1}
	}
	if shape.Sensor() {
		return cp.FColor{R: 1.0, G: 0.85, B: 0.2, A: 1.0}
	}
	if shape.Body() != nil && shape.Body().GetType() == cp.BODY_STATIC {
		return cp.FColor{R: 0.4, G: 0.7, B: 1.0, A: 1.0}
	}
	return cp.FColor{R: 0.9, G: 0.4, B: 0.9, A: 1.0}
}

func (d *chipmunkDrawer) ConstraintColor() cp.FColor {
	return cp.FColor{R: 0.7, G: 0.7, B: 0.7, A: 1.0}
}

func (d *chipmunkDrawer) CollisionPointColor() cp.FColor {
	return cp.FColor{R: 1.0, G: 0.1, B: 0.1, A: 1.0}
}

func (d *chipmunkDrawer) Data() interface{} {
	return nil
}

func fcolorToRGBA(c cp.FColor) color.RGBA {
	clamp := func(v float32) uint8 {
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		return uint8(v * 255)
	}
	return color.RGBA{R: clamp(c.R), G: clamp(c.G), B: clamp(c.B), A: clamp(c.A)}
}
