package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func closestAutoAnchorTarget(w *ecs.World, player ecs.Entity, startX, startY, minDistance, maxDistance float64) (float64, float64, bool) {
	if w == nil || maxDistance <= 0 {
		return 0, 0, false
	}
	if minDistance < 0 {
		minDistance = 0
	}
	if minDistance > maxDistance {
		return 0, 0, false
	}
	originX, originY := startX, startY
	if playerTransform, ok := ecs.Get(w, player, component.TransformComponent.Kind()); ok {
		if playerBody, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind()); ok {
			if bodyX, bodyY, ok := physicsBodyCenter(w, player, playerTransform, playerBody); ok {
				originX = bodyX
				originY = bodyY
			}
		}
	}

	bestDistance := maxDistance + 1
	bestX := 0.0
	bestY := 0.0
	found := false

	considerCandidate := func(candidateX, candidateY float64) {
		dx := candidateX - originX
		dy := candidateY - originY
		rawDistance := math.Hypot(dx, dy)
		if rawDistance <= 0 || rawDistance < minDistance || rawDistance > maxDistance || rawDistance >= bestDistance {
			return
		}

		traceScale := (rawDistance + 0.5) / rawDistance
		traceEndX := originX + dx*traceScale
		traceEndY := originY + dy*traceScale
		hitX, hitY, ok, valid := firstStaticHit(w, player, originX, originY, traceEndX, traceEndY)
		if !ok || !valid {
			return
		}

		hitDistance := math.Hypot(hitX-originX, hitY-originY)
		if hitDistance <= 0 || hitDistance < minDistance || hitDistance > maxDistance || hitDistance >= bestDistance {
			return
		}

		bestDistance = hitDistance
		bestX = hitX
		bestY = hitY
		found = true
	}

	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, body *component.PhysicsBody, transform *component.Transform) {
		if e == player || !body.Static || ecs.Has(w, e, component.SpikeTagComponent.Kind()) {
			return
		}

		if body.Radius > 0 {
			centerX := bodyCenterX(w, e, transform, &component.PhysicsBody{
				OffsetX:      body.OffsetX,
				OffsetY:      body.OffsetY,
				Width:        2 * body.Radius,
				Height:       2 * body.Radius,
				AlignTopLeft: body.AlignTopLeft,
			})
			centerY := bodyCenterY(transform, &component.PhysicsBody{
				OffsetX:      body.OffsetX,
				OffsetY:      body.OffsetY,
				Width:        2 * body.Radius,
				Height:       2 * body.Radius,
				AlignTopLeft: body.AlignTopLeft,
			})
			considerCandidate(closestPointOnCircle(originX, originY, centerX, centerY, body.Radius))
			return
		}

		minX, minY, maxX, maxY := bodyAABB(w, e, transform, body)
		considerCandidate(closestPointOnAABB(originX, originY, minX, minY, maxX, maxY))
	})

	return bestX, bestY, found
}

func closestPointOnAABB(x, y, minX, minY, maxX, maxY float64) (float64, float64) {
	clampedX := math.Max(minX, math.Min(x, maxX))
	clampedY := math.Max(minY, math.Min(y, maxY))
	inside := x >= minX && x <= maxX && y >= minY && y <= maxY
	if !inside {
		return clampedX, clampedY
	}

	leftDist := math.Abs(x - minX)
	rightDist := math.Abs(maxX - x)
	topDist := math.Abs(y - minY)
	bottomDist := math.Abs(maxY - y)

	closestX, closestY := minX, y
	closestDist := leftDist
	if rightDist < closestDist {
		closestDist = rightDist
		closestX, closestY = maxX, y
	}
	if topDist < closestDist {
		closestDist = topDist
		closestX, closestY = x, minY
	}
	if bottomDist < closestDist {
		closestX, closestY = x, maxY
	}
	return closestX, closestY
}

func closestPointOnCircle(x, y, centerX, centerY, radius float64) (float64, float64) {
	dx := x - centerX
	dy := y - centerY
	distance := math.Hypot(dx, dy)
	if distance == 0 {
		return centerX + radius, centerY
	}
	scale := radius / distance
	return centerX + dx*scale, centerY + dy*scale
}
