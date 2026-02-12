package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func firstStaticHit(w *ecs.World, player ecs.Entity, x0, y0, x1, y1 float64) (float64, float64, bool) {
	if w == nil {
		return 0, 0, false
	}

	dx := x1 - x0
	dy := y1 - y0
	if dx == 0 && dy == 0 {
		return 0, 0, false
	}

	closestT := 1.0
	hasHit := false

	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, body *component.PhysicsBody, transform *component.Transform) {
		if e == player || !body.Static {
			return
		}

		if body.Radius > 0 {
			x, y, ok := segmentCircleHit(x0, y0, x1, y1, transform, body)
			if ok {
				t := hitParam(x0, y0, x1, y1, x, y)
				if t >= 0 && t < closestT {
					closestT = t
					hasHit = true
				}
			}
			return
		}

		minX, minY, maxX, maxY := bodyAABB(transform, body)
		if hit, t := segmentAABBHit(x0, y0, dx, dy, minX, minY, maxX, maxY); hit {
			if t >= 0 && t < closestT {
				closestT = t
				hasHit = true
			}
		}
	})

	if !hasHit {
		return 0, 0, false
	}

	return x0 + dx*closestT, y0 + dy*closestT, true
}

func bodyAABB(transform *component.Transform, body *component.PhysicsBody) (minX, minY, maxX, maxY float64) {
	width := body.Width
	height := body.Height
	if width <= 0 {
		width = 32
	}
	if height <= 0 {
		height = 32
	}

	if body.AlignTopLeft {
		minX = transform.X + body.OffsetX
		minY = transform.Y + body.OffsetY
	} else {
		minX = transform.X + body.OffsetX - width/2
		minY = transform.Y + body.OffsetY - height/2
	}
	maxX = minX + width
	maxY = minY + height
	return
}

func segmentAABBHit(x0, y0, dx, dy, minX, minY, maxX, maxY float64) (bool, float64) {
	tmin := 0.0
	tmax := 1.0

	if dx != 0 {
		invD := 1.0 / dx
		t1 := (minX - x0) * invD
		t2 := (maxX - x0) * invD
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tmin = math.Max(tmin, t1)
		tmax = math.Min(tmax, t2)
	} else if x0 < minX || x0 > maxX {
		return false, 0
	}

	if dy != 0 {
		invD := 1.0 / dy
		t1 := (minY - y0) * invD
		t2 := (maxY - y0) * invD
		if t1 > t2 {
			t1, t2 = t2, t1
		}
		tmin = math.Max(tmin, t1)
		tmax = math.Min(tmax, t2)
	} else if y0 < minY || y0 > maxY {
		return false, 0
	}

	if tmax >= tmin {
		return true, tmin
	}
	return false, 0
}

func segmentCircleHit(x0, y0, x1, y1 float64, transform *component.Transform, body *component.PhysicsBody) (float64, float64, bool) {
	r := body.Radius
	if r <= 0 {
		return 0, 0, false
	}

	centerX := transform.X + body.OffsetX
	centerY := transform.Y + body.OffsetY
	if body.AlignTopLeft {
		centerX = transform.X + body.OffsetX + r
		centerY = transform.Y + body.OffsetY + r
	}

	dx := x1 - x0
	dy := y1 - y0
	fx := x0 - centerX
	fy := y0 - centerY

	a := dx*dx + dy*dy
	b := 2 * (fx*dx + fy*dy)
	c := fx*fx + fy*fy - r*r

	disc := b*b - 4*a*c
	if disc < 0 || a == 0 {
		return 0, 0, false
	}

	sqrtDisc := math.Sqrt(disc)
	t1 := (-b - sqrtDisc) / (2 * a)
	t2 := (-b + sqrtDisc) / (2 * a)

	t := math.Inf(1)
	if t1 >= 0 && t1 <= 1 {
		t = t1
	}
	if t2 >= 0 && t2 <= 1 && t2 < t {
		t = t2
	}
	if !math.IsInf(t, 1) {
		return x0 + dx*t, y0 + dy*t, true
	}
	return 0, 0, false
}

func hitParam(x0, y0, x1, y1, hx, hy float64) float64 {
	dx := x1 - x0
	dy := y1 - y0
	if math.Abs(dx) > math.Abs(dy) {
		if dx == 0 {
			return 0
		}
		return (hx - x0) / dx
	}
	if dy == 0 {
		return 0
	}
	return (hy - y0) / dy
}
