package module

import (
	"fmt"
	"math"

	"github.com/d5/tengo/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func PhysicsModule() Module {
	return Module{
		Name: "physics",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["set_disabled"] = &tengo.UserFunction{Name: "set_disabled", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("set_disabled requires 1 argument: boolean value")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				physicsBody.Disabled = objectAsBool(args[0])
				if physicsBody.Disabled {
					physicsBody.Body = nil
					physicsBody.Shape = nil
				}

				return tengo.TrueValue, nil
			}}

			// sig: stop_x() -> bool
			// doc: Stop horizontal movement on the entity's physics body.
			values["stop_x"] = &tengo.UserFunction{Name: "stop_x", Value: func(args ...tengo.Object) (tengo.Object, error) {
				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				physicsBody.Body.SetVelocity(0, physicsBody.Body.Velocity().Y)
				return tengo.TrueValue, nil
			}}

			values["stop_xy"] = &tengo.UserFunction{Name: "stop_xy", Value: func(args ...tengo.Object) (tengo.Object, error) {
				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				physicsBody.Body.SetVelocity(0, 0)

				return tengo.TrueValue, nil
			}}

			values["has_down_surface"] = &tengo.UserFunction{Name: "has_down_surface", Value: func(args ...tengo.Object) (tengo.Object, error) {
				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody == nil || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || transform == nil {
					return tengo.FalseValue, fmt.Errorf("Transform component not found for entity %v", target)
				}

				hasHit, validHit, err := hasAnyDownSurface(world, target, transform, physicsBody, args)
				if err != nil {
					return tengo.FalseValue, err
				}
				if !hasHit || !validHit {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			values["snap_to_down_surface"] = &tengo.UserFunction{Name: "snap_to_down_surface", Value: func(args ...tengo.Object) (tengo.Object, error) {
				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody == nil || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || transform == nil {
					return tengo.FalseValue, fmt.Errorf("Transform component not found for entity %v", target)
				}

				hitX, hitY, hasHit, validHit, downX, downY, faceDistance, err := downSurfaceProbe(world, target, transform, physicsBody, args)
				if err != nil {
					return tengo.FalseValue, err
				}
				if !hasHit || !validHit {
					return tengo.FalseValue, nil
				}

				desiredCenterX := hitX - downX*faceDistance
				desiredCenterY := hitY - downY*faceDistance

				physicsBody.Body.SetPosition(cp.Vector{X: desiredCenterX, Y: desiredCenterY})
				syncTransformToBody(world, target, transform, physicsBody, desiredCenterX, desiredCenterY)

				return tengo.TrueValue, nil
			}}

			// sig: jump(force float) -> bool
			// doc: Apply an upwards impulse to make the entity jump; returns true when applied.
			values["jump"] = &tengo.UserFunction{Name: "jump", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("jump requires 1 argument: jump velocity")
				}

				height := objectAsFloat(args[0])
				if height < 0 {
					return tengo.FalseValue, fmt.Errorf("jump velocity must be non-negative")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				physicsBody.Body.SetVelocity(physicsBody.Body.Velocity().X, -height)

				return tengo.TrueValue, nil
			}}

			// sig: is_grounded() -> bool
			// doc: Returns true if the entity is currently touching the ground.
			values["is_grounded"] = &tengo.UserFunction{Name: "is_grounded", Value: func(args ...tengo.Object) (tengo.Object, error) {
				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				transform, _ := ecs.Get(world, target, component.TransformComponent.Kind())

				// Prefer to use the physics body position when available so the
				// probe originates from the actual body center. Fall back to the
				// transform position otherwise.
				px := transform.X
				py := transform.Y
				if physicsBody.Body != nil {
					p := physicsBody.Body.Position()
					px = p.X
					py = p.Y
				}

				probeDist := 8.0
				if physicsBody.Height > 0 {
					probeDist = physicsBody.Height/2 + 2
				}

				_, _, hit, _ := firstStaticHit(world, target, px, py, px, py+probeDist)
				if !hit {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func hasAnyDownSurface(world *ecs.World, target ecs.Entity, transform *component.Transform, physicsBody *component.PhysicsBody, args []tengo.Object) (bool, bool, error) {
	downX, downY, faceDistance, probeDistance, err := downSurfaceParams(world, target, transform, physicsBody, args)
	if err != nil {
		return false, false, err
	}
	if math.Abs(downX) < 1e-9 && math.Abs(downY) < 1e-9 {
		return false, false, nil
	}

	center := physicsBody.Body.Position()
	tangentX := downY
	tangentY := -downX
	tangentDistance := surfaceFaceDistance(physicsBody, tangentX, tangentY)
	probeLength := faceDistance + probeDistance

	probeOffsets := downSurfaceProbeOffsets(tangentDistance)

	anyHit := false
	anyValidHit := false
	for _, offset := range probeOffsets {
		originX := center.X + tangentX*offset
		originY := center.Y + tangentY*offset
		_, _, hasHit, validHit := firstStaticHit(
			world,
			target,
			originX,
			originY,
			originX+downX*probeLength,
			originY+downY*probeLength,
		)
		if hasHit {
			anyHit = true
			if validHit {
				anyValidHit = true
			}
		}
	}

	return anyHit, anyValidHit, nil
}

func downSurfaceProbe(world *ecs.World, target ecs.Entity, transform *component.Transform, physicsBody *component.PhysicsBody, args []tengo.Object) (hitX, hitY float64, hasHit, validHit bool, downX, downY, faceDistance float64, err error) {
	downX, downY, faceDistance, probeDistance, err := downSurfaceParams(world, target, transform, physicsBody, args)
	if err != nil {
		return 0, 0, false, false, 0, 0, 0, err
	}
	if math.Abs(downX) < 1e-9 && math.Abs(downY) < 1e-9 {
		return 0, 0, false, false, 0, 0, 0, nil
	}

	center := physicsBody.Body.Position()
	tangentX := downY
	tangentY := -downX
	tangentDistance := surfaceFaceDistance(physicsBody, tangentX, tangentY)
	probeLength := faceDistance + probeDistance
	bestT := math.Inf(1)
	hasAnyHit := false
	hasAnyValidHit := false
	invalidHitX := 0.0
	invalidHitY := 0.0

	for _, offset := range downSurfaceProbeOffsets(tangentDistance) {
		originX := center.X + tangentX*offset
		originY := center.Y + tangentY*offset
		endX := originX + downX*probeLength
		endY := originY + downY*probeLength

		candidateX, candidateY, candidateHit, candidateValid := firstStaticHit(
			world,
			target,
			originX,
			originY,
			endX,
			endY,
		)
		if !candidateHit {
			continue
		}

		hasAnyHit = true
		if !candidateValid {
			if !hasAnyValidHit {
				invalidHitX = candidateX
				invalidHitY = candidateY
			}
			continue
		}

		hasAnyValidHit = true
		candidateT := hitParam(originX, originY, endX, endY, candidateX, candidateY)
		if candidateT < bestT {
			bestT = candidateT
			hitX = candidateX
			hitY = candidateY
		}
	}

	if hasAnyValidHit {
		return hitX, hitY, true, true, downX, downY, faceDistance, nil
	}
	if hasAnyHit {
		return invalidHitX, invalidHitY, true, false, downX, downY, faceDistance, nil
	}

	return 0, 0, false, false, downX, downY, faceDistance, nil
}

func downSurfaceProbeOffsets(tangentDistance float64) []float64 {
	if tangentDistance <= 1 {
		return []float64{0}
	}
	edgeOffset := math.Max(0, tangentDistance-1)
	if edgeOffset <= 0 {
		return []float64{0}
	}
	halfOffset := edgeOffset * 0.5
	return []float64{0, -halfOffset, halfOffset, -edgeOffset, edgeOffset}
}

func downSurfaceParams(world *ecs.World, target ecs.Entity, transform *component.Transform, physicsBody *component.PhysicsBody, args []tengo.Object) (downX, downY, faceDistance, probeDistance float64, err error) {
	rotation := scriptRotationRadians(world, target, transform)
	downX = -math.Sin(rotation)
	downY = math.Cos(rotation)

	probeDistance = math.Max(physicsBody.Width, physicsBody.Height) + 4
	if physicsBody.Radius > 0 {
		probeDistance = physicsBody.Radius + 4
	}
	if len(args) >= 1 {
		probeDistance = objectAsFloat(args[0])
	}
	if probeDistance < 0 {
		return 0, 0, 0, 0, fmt.Errorf("probe distance must be non-negative")
	}

	faceDistance = surfaceFaceDistance(physicsBody, downX, downY)
	return downX, downY, faceDistance, probeDistance, nil
}

func surfaceFaceDistance(body *component.PhysicsBody, downX, downY float64) float64 {
	if body == nil {
		return 16
	}
	if body.Radius > 0 {
		return body.Radius
	}
	halfWidth := body.Width / 2
	halfHeight := body.Height / 2
	if halfWidth <= 0 {
		halfWidth = 16
	}
	if halfHeight <= 0 {
		halfHeight = 16
	}

	return math.Abs(downX)*halfWidth + math.Abs(downY)*halfHeight
}

func syncTransformToBody(world *ecs.World, target ecs.Entity, transform *component.Transform, body *component.PhysicsBody, centerX, centerY float64) {
	if transform == nil || body == nil {
		return
	}

	effectiveOffsetX := facingAdjustedOffsetX(world, target, body.OffsetX, body.Width, body.AlignTopLeft)
	if body.AlignTopLeft {
		transform.X = centerX - body.Width/2 - effectiveOffsetX
		transform.Y = centerY - body.Height/2 - body.OffsetY
	} else {
		transform.X = centerX - effectiveOffsetX
		transform.Y = centerY - body.OffsetY
	}

	if body.Body != nil {
		transform.Rotation = body.Body.Angle()
	}
}
