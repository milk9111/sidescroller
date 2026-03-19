package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

// firstStaticHit casts a line segment from (x0,y0) to (x1,y1) against
// static collision geometry and hazards in the world `w`, skipping the
// `ent` entity. It returns the hit point (x,y), a boolean indicating
// whether any hit occurred, and a boolean indicating whether the hit is a
// valid anchor surface (true for ordinary static bodies, false for spikes
// or hazardous surfaces).
//
// The function checks physics bodies as either circles (when `Radius>0`)
// or AABBs and also tests spike/hazard bounds. The first intersection
// along the segment is returned (closest to the start point).
func firstStaticHit(w *ecs.World, ent ecs.Entity, x0, y0, x1, y1 float64) (float64, float64, bool, bool) {
	if w == nil {
		return 0, 0, false, false
	}

	dx := x1 - x0
	dy := y1 - y0
	if dx == 0 && dy == 0 {
		return 0, 0, false, false
	}

	closestT := 1.0
	hasHit := false
	hitValid := false

	considerHit := func(t float64, valid bool) {
		if t < 0 || t >= closestT {
			return
		}
		closestT = t
		hasHit = true
		hitValid = valid
	}

	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, body *component.PhysicsBody, transform *component.Transform) {
		if e == ent || !body.Static || body.Disabled {
			return
		}
		validAnchorSurface := !ecs.Has(w, e, component.SpikeTagComponent.Kind())

		if body.Radius > 0 {
			x, y, ok := segmentCircleHit(w, e, x0, y0, x1, y1, transform, body)
			if ok {
				t := hitParam(x0, y0, x1, y1, x, y)
				considerHit(t, validAnchorSurface)
			}
			return
		}

		minX, minY, maxX, maxY := bodyAABB(w, e, transform, body)
		if hit, t := segmentAABBHit(x0, y0, dx, dy, minX, minY, maxX, maxY); hit {
			considerHit(t, validAnchorSurface)
		}
	})

	ecs.ForEach3(w, component.SpikeTagComponent.Kind(), component.HazardComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, _ *component.SpikeTag, h *component.Hazard, t *component.Transform) {
		if e == ent {
			return
		}
		bounds, ok := hazardBounds(w, e, h, t)
		if !ok {
			return
		}
		if hit, t := segmentAABBHit(x0, y0, dx, dy, bounds.x, bounds.y, bounds.x+bounds.w, bounds.y+bounds.h); hit {
			considerHit(t, false)
		}
	})

	if !hasHit {
		return 0, 0, false, false
	}

	return x0 + dx*closestT, y0 + dy*closestT, true, hitValid
}

func bodyAABB(w *ecs.World, e ecs.Entity, transform *component.Transform, body *component.PhysicsBody) (minX, minY, maxX, maxY float64) {
	minX, minY, maxX, maxY, _ = physicsBodyBounds(w, e, transform, body)
	return
}

// segmentAABBHit tests a line segment starting at (x0,y0) with vector
// (dx,dy) against an axis-aligned bounding box defined by
// [minX,minY] - [maxX,maxY]. It returns (true, t) when the segment
// intersects the box, where `t` is the param along the segment in [0,1]
// for the first intersection point (so intersection point = x0+dx*t,
// y0+dy*t). If there is no intersection it returns (false,0).
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

func segmentCircleHit(w *ecs.World, e ecs.Entity, x0, y0, x1, y1 float64, transform *component.Transform, body *component.PhysicsBody) (float64, float64, bool) {
	r := body.Radius
	if r <= 0 {
		return 0, 0, false
	}

	centerX := bodyCenterX(w, e, transform, &component.PhysicsBody{OffsetX: body.OffsetX, OffsetY: body.OffsetY, Width: 2 * r, Height: 2 * r, AlignTopLeft: body.AlignTopLeft})
	centerY := bodyCenterY(transform, &component.PhysicsBody{OffsetX: body.OffsetX, OffsetY: body.OffsetY, Width: 2 * r, Height: 2 * r, AlignTopLeft: body.AlignTopLeft})

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
