package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const autoAnchorHorizontalEpsilon = 0.001
const autoAnchorSurfaceInset = 0.5

type autoAnchorCandidate struct {
	x float64
	y float64
}

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

	horizontalPreference := autoAnchorHorizontalPreference(w, player)
	bestForward := math.Inf(-1)
	bestDistance := maxDistance + 1
	bestX := 0.0
	bestY := 0.0
	found := false

	considerCandidate := func(candidateX, candidateY float64) {
		dx := candidateX - originX
		dy := candidateY - originY
		rawDistance := math.Hypot(dx, dy)
		if rawDistance <= 0 || rawDistance < minDistance || rawDistance > maxDistance {
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
		if hitDistance <= 0 || hitDistance < minDistance || hitDistance > maxDistance {
			return
		}

		forward := horizontalPreference * (hitX - originX)
		if !shouldReplaceAutoAnchorCandidate(horizontalPreference, forward, hitDistance, bestForward, bestDistance, found) {
			return
		}

		bestForward = forward
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
		for _, candidate := range autoAnchorAABBCandidates(originX, originY, minX, minY, maxX, maxY, horizontalPreference, maxDistance) {
			considerCandidate(candidate.x, candidate.y)
		}
	})

	return bestX, bestY, found
}

func autoAnchorHorizontalPreference(w *ecs.World, player ecs.Entity) float64 {
	if w == nil {
		return 0
	}
	if input, ok := ecs.Get(w, player, component.InputComponent.Kind()); ok && input != nil {
		if input.MoveX > autoAnchorHorizontalEpsilon {
			return 1
		}
		if input.MoveX < -autoAnchorHorizontalEpsilon {
			return -1
		}
	}
	if body, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind()); ok && body != nil && body.Body != nil {
		vx := body.Body.Velocity().X
		if vx > autoAnchorHorizontalEpsilon {
			return 1
		}
		if vx < -autoAnchorHorizontalEpsilon {
			return -1
		}
	}
	return 0
}

func shouldReplaceAutoAnchorCandidate(horizontalPreference, forward, distance, bestForward, bestDistance float64, found bool) bool {
	if !found {
		return true
	}
	if math.Abs(horizontalPreference) > autoAnchorHorizontalEpsilon {
		if forward > bestForward+autoAnchorHorizontalEpsilon {
			return true
		}
		if forward < bestForward-autoAnchorHorizontalEpsilon {
			return false
		}
	}
	return distance < bestDistance-autoAnchorHorizontalEpsilon
}

func autoAnchorAABBCandidates(originX, originY, minX, minY, maxX, maxY, horizontalPreference, maxDistance float64) []autoAnchorCandidate {
	candidates := make([]autoAnchorCandidate, 0, 8)

	clampedX := clampAutoAnchor(originX, minX, maxX)
	clampedY := clampAutoAnchor(originY, minY, maxY)
	candidates = append(candidates,
		autoAnchorCandidate{x: clampedX, y: minY},
		autoAnchorCandidate{x: clampedX, y: maxY},
		autoAnchorCandidate{x: minX, y: clampedY},
		autoAnchorCandidate{x: maxX, y: clampedY},
		candidateFromPoint(closestPointOnAABB(originX, originY, minX, minY, maxX, maxY)),
	)

	if math.Abs(horizontalPreference) > autoAnchorHorizontalEpsilon {
		for _, edgeY := range []float64{minY, maxY} {
			if candidate, ok := autoAnchorForwardEdgeCandidate(originX, originY, edgeY, minX, maxX, horizontalPreference, maxDistance); ok {
				candidates = append(candidates, candidate)
			}
		}
		candidates = append(candidates, autoAnchorCandidate{x: preferredAABBEdgeX(horizontalPreference, minX, maxX), y: insetVerticalEdgeY(clampedY, minY, maxY)})
	}

	return dedupeAutoAnchorCandidates(candidates)
}

func autoAnchorForwardEdgeCandidate(originX, originY, edgeY, minX, maxX, horizontalPreference, maxDistance float64) (autoAnchorCandidate, bool) {
	if maxDistance <= 0 {
		return autoAnchorCandidate{}, false
	}
	dy := edgeY - originY
	remaining := maxDistance*maxDistance - dy*dy
	if remaining < 0 {
		return autoAnchorCandidate{}, false
	}
	reachX := originX + horizontalPreference*math.Sqrt(remaining)
	return autoAnchorCandidate{x: insetHorizontalEdgeX(clampAutoAnchor(reachX, minX, maxX), minX, maxX), y: edgeY}, true
}

func preferredAABBEdgeX(horizontalPreference, minX, maxX float64) float64 {
	if horizontalPreference < 0 {
		return minX
	}
	return maxX
}

func dedupeAutoAnchorCandidates(candidates []autoAnchorCandidate) []autoAnchorCandidate {
	result := make([]autoAnchorCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		duplicate := false
		for _, existing := range result {
			if math.Abs(existing.x-candidate.x) <= autoAnchorHorizontalEpsilon && math.Abs(existing.y-candidate.y) <= autoAnchorHorizontalEpsilon {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, candidate)
		}
	}
	return result
}

func candidateFromPoint(x, y float64) autoAnchorCandidate {
	return autoAnchorCandidate{x: x, y: y}
}

func clampAutoAnchor(value, minValue, maxValue float64) float64 {
	return math.Max(minValue, math.Min(value, maxValue))
}

func insetHorizontalEdgeX(x, minX, maxX float64) float64 {
	if maxX-minX <= autoAnchorSurfaceInset*2 {
		return (minX + maxX) / 2
	}
	return clampAutoAnchor(x, minX+autoAnchorSurfaceInset, maxX-autoAnchorSurfaceInset)
}

func insetVerticalEdgeY(y, minY, maxY float64) float64 {
	if maxY-minY <= autoAnchorSurfaceInset*2 {
		return (minY + maxY) / 2
	}
	return clampAutoAnchor(y, minY+autoAnchorSurfaceInset, maxY-autoAnchorSurfaceInset)
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
