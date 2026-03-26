package module

import (
	"fmt"
	"math"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func AIModule() Module {
	return Module{
		Name: "ai",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["move_forward"] = &tengo.UserFunction{Name: "move_forward", Value: func(args ...tengo.Object) (tengo.Object, error) {
				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Sprite component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				dx := ai.MoveSpeed
				if len(args) >= 1 {
					dx = objectAsFloat(args[0])
				}

				forward := 1
				if sprite.FacingLeft {
					forward = -1
				}

				rotation := scriptRotationRadians(world, target, transform)

				baseForwardX, baseForwardY := scriptForwardVector(rotation)
				forwardX := baseForwardX * float64(forward)
				forwardY := baseForwardY * float64(forward)

				lateralX := -forwardY
				lateralY := forwardX

				velocity := physicsBody.Body.Velocity()
				lateralSpeed := velocity.X*lateralX + velocity.Y*lateralY

				physicsBody.Body.SetVelocity(
					forwardX*dx+lateralX*lateralSpeed,
					forwardY*dx+lateralY*lateralSpeed,
				)

				return tengo.TrueValue, nil
			}}

			values["move_forward_transform"] = &tengo.UserFunction{Name: "move_forward_transform", Value: func(args ...tengo.Object) (tengo.Object, error) {
				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Sprite component not found")
				}

				dx := ai.MoveSpeed
				if len(args) >= 1 {
					dx = objectAsFloat(args[0])
				}

				forward := 1
				if sprite.FacingLeft {
					forward = -1
				}

				rotation := scriptRotationRadians(world, target, transform)
				baseForwardX, baseForwardY := scriptForwardVector(rotation)
				forwardX := baseForwardX * float64(forward)
				forwardY := baseForwardY * float64(forward)

				transform.X += forwardX * dx
				transform.Y += forwardY * dx

				return tengo.TrueValue, nil
			}}

			values["sees_player"] = &tengo.UserFunction{Name: "sees_player", Value: func(args ...tengo.Object) (tengo.Object, error) {
				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				playerTransform, ok := ecs.Get(world, playerEnt, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player Transform component not found")
				}

				pos := physicsBody.Body.Position()

				dx := playerTransform.X - pos.X
				dy := playerTransform.Y - pos.Y
				if math.Hypot(dx, dy) > ai.FollowRange {
					return tengo.FalseValue, nil
				}

				_, _, hasHit, _ := firstStaticHit(world, target, pos.X, pos.Y, playerTransform.X, playerTransform.Y)
				if hasHit {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			values["player_in_range"] = &tengo.UserFunction{Name: "player_in_range", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("player_in_range requires 1 argument: the range to check")
				}

				rng := objectAsFloat(args[0])
				if rng < 0 {
					return tengo.FalseValue, fmt.Errorf("range must be non-negative")
				}

				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				var x, y float64

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
					if !ok {
						return tengo.FalseValue, fmt.Errorf("PhysicsBody or Transform component not found")
					}

					x, y = transform.X, transform.Y
				} else {
					pos := physicsBody.Body.Position()
					x, y = pos.X, pos.Y
				}

				playerPhysicsBody, ok := ecs.Get(world, playerEnt, component.PhysicsBodyComponent.Kind())
				if !ok || playerPhysicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("Player PhysicsBody component not found")
				}

				dist := math.Hypot(playerPhysicsBody.Body.Position().X-x, playerPhysicsBody.Body.Position().Y-y)
				// fmt.Println("player_in_range: distance to player is", dist)
				if dist > rng {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			values["player_position"] = &tengo.UserFunction{Name: "player_position", Value: func(args ...tengo.Object) (tengo.Object, error) {
				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, fmt.Errorf("Player not found")
				}

				x, y, err := scriptEntityPosition(world, playerEnt)
				if err != nil {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, err
				}

				return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: x}, &tengo.Float{Value: y}}}, nil
			}}

			values["direction_to_point_2d"] = &tengo.UserFunction{Name: "direction_to_point_2d", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, fmt.Errorf("direction_to_point_2d requires 2 arguments: x and y")
				}

				targetX := objectAsFloat(args[0])
				targetY := objectAsFloat(args[1])

				currentX, currentY, err := scriptEntityPosition(world, target)
				if err != nil {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, err
				}

				dx := targetX - currentX
				dy := targetY - currentY
				dist := math.Hypot(dx, dy)
				if dist < 1e-4 {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, nil
				}

				return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: dx / dist}, &tengo.Float{Value: dy / dist}}}, nil
			}}

			values["move_direction_2d"] = &tengo.UserFunction{Name: "move_direction_2d", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("move_direction_2d requires at least 2 arguments: x and y")
				}

				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				dirX := objectAsFloat(args[0])
				dirY := objectAsFloat(args[1])
				speed := ai.MoveSpeed
				if len(args) >= 3 {
					speed = objectAsFloat(args[2])
				}

				mag := math.Hypot(dirX, dirY)
				if mag < 1e-4 {
					physicsBody.Body.SetVelocity(0, 0)
					return tengo.TrueValue, nil
				}

				physicsBody.Body.SetVelocity(dirX/mag*speed, dirY/mag*speed)

				return tengo.TrueValue, nil
			}}

			values["blocked_in_direction_2d"] = &tengo.UserFunction{Name: "blocked_in_direction_2d", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("blocked_in_direction_2d requires at least 2 arguments: x and y")
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || transform == nil {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody == nil || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				dirX := objectAsFloat(args[0])
				dirY := objectAsFloat(args[1])
				probeDistance := 8.0
				if len(args) >= 3 {
					probeDistance = objectAsFloat(args[2])
				}
				if probeDistance < 0 {
					return tengo.FalseValue, fmt.Errorf("probe distance must be non-negative")
				}

				mag := math.Hypot(dirX, dirY)
				if mag < 1e-4 {
					return tengo.FalseValue, nil
				}

				dirX /= mag
				dirY /= mag

				centerX, centerY, err := scriptEntityPosition(world, target)
				if err != nil {
					return tengo.FalseValue, err
				}

				minX, minY, maxX, maxY := bodyAABB(world, target, transform, physicsBody)
				faceDistance := math.Abs(dirX)*(maxX-minX)/2 + math.Abs(dirY)*(maxY-minY)/2
				if faceDistance <= 0 {
					faceDistance = 16
				}

				tangentX := -dirY
				tangentY := dirX
				tangentDistance := math.Abs(tangentX)*(maxX-minX)/2 + math.Abs(tangentY)*(maxY-minY)/2
				edgeOffset := math.Max(0, tangentDistance-1)
				probeOffsets := []float64{0}
				if edgeOffset > 0 {
					probeOffsets = append(probeOffsets, edgeOffset, -edgeOffset)
				}

				hasHit := false
				for _, offset := range probeOffsets {
					originX := centerX + tangentX*offset
					originY := centerY + tangentY*offset

					_, _, probeHit, _ := firstStaticHit(
						world,
						target,
						originX,
						originY,
						originX+dirX*(faceDistance+probeDistance),
						originY+dirY*(faceDistance+probeDistance),
					)

					if probeHit {
						hasHit = true
						break
					}
				}

				return boolObject(hasHit), nil
			}}

			values["move_towards_point_2d"] = &tengo.UserFunction{Name: "move_towards_point_2d", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("move_towards_point_2d requires at least 2 arguments: x and y")
				}

				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				targetX := objectAsFloat(args[0])
				targetY := objectAsFloat(args[1])
				speed := ai.MoveSpeed
				if len(args) >= 3 {
					speed = objectAsFloat(args[2])
				}
				stopDistance := 0.0
				if len(args) >= 4 {
					stopDistance = objectAsFloat(args[3])
				}

				currentX, currentY, err := scriptEntityPosition(world, target)
				if err != nil {
					return tengo.FalseValue, err
				}

				dx := targetX - currentX
				dy := targetY - currentY
				dist := math.Hypot(dx, dy)
				if dist < 1e-4 || dist <= stopDistance {
					physicsBody.Body.SetVelocity(0, 0)
					return tengo.TrueValue, nil
				}

				physicsBody.Body.SetVelocity(dx/dist*speed, dy/dist*speed)

				return tengo.TrueValue, nil
			}}

			values["move_towards_player"] = &tengo.UserFunction{Name: "move_towards_player", Value: func(args ...tengo.Object) (tengo.Object, error) {
				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				playerPhysicsBody, ok := ecs.Get(world, playerEnt, component.PhysicsBodyComponent.Kind())
				if !ok || playerPhysicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("Player PhysicsBody component not found")
				}

				pos := physicsBody.Body.Position()

				dx := playerPhysicsBody.Body.Position().X - pos.X
				dy := playerPhysicsBody.Body.Position().Y - pos.Y

				stopDistance := ai.AttackRange
				if stopDistance < 24 {
					stopDistance = 24
				}

				horizontalDeadzone := 4.0
				if math.Abs(dy) > 24 {
					horizontalDeadzone = 10.0
				}

				dir := 0.0
				if math.Abs(dx) > horizontalDeadzone && math.Abs(dx) > stopDistance {
					if dx > 0 {
						dir = 1
					} else {
						dir = -1
					}
				}

				const desiredSeparation = 40.0
				const verticalNeighborBand = 40.0
				const maxRepel = 1.0
				const repelWeight = 0.9

				repelX := 0.0
				repelY := 0.0

				ecs.ForEach3(world,
					component.AITagComponent.Kind(),
					component.PhysicsBodyComponent.Kind(),
					component.TransformComponent.Kind(),
					func(other ecs.Entity, _ *component.AITag, ob *component.PhysicsBody, ot *component.Transform) {
						if other == target {
							return
						}

						// Determine neighbor position (prefer physics body)
						nx, ny := 0.0, 0.0
						if ob != nil && ob.Body != nil {
							p := ob.Body.Position()
							nx, ny = p.X, p.Y
						} else if ot != nil {
							nx, ny = ot.X, ot.Y
						} else {
							return
						}

						// Only consider neighbors roughly on the same platform level
						if math.Abs(ny-pos.Y) > verticalNeighborBand {
							return
						}

						dx := pos.X - nx
						dy := pos.Y - ny
						dist := math.Hypot(dx, dy)
						if dist < 0.001 || dist >= desiredSeparation {
							return
						}

						// stronger push when very close, smooth to zero at desiredSeparation
						strength := (desiredSeparation - dist) / desiredSeparation
						// normalized direction from neighbor to self
						nxDir := dx / dist
						nyDir := dy / dist
						repelX += nxDir * strength
						repelY += nyDir * strength
					},
				)

				// apply only horizontal component to movement, cap magnitude
				mag := math.Hypot(repelX, repelY)
				if mag > 0.0001 {
					if mag > maxRepel {
						repelX = (repelX / mag) * maxRepel
					}
					// apply horizontal influence scaled by weight
					dir += repelX * repelWeight
					if dir > 1 {
						dir = 1
					} else if dir < -1 {
						dir = -1
					}
					if math.Abs(dir) < 0.15 {
						dir = 0
					}
				}

				if dir != 0 {
					if nav, ok := ecs.Get(world, target, component.AINavigationComponent.Kind()); ok && nav != nil {
						if dir > 0 && !nav.GroundAheadRight {
							dir = 0
						} else if dir < 0 && !nav.GroundAheadLeft {
							dir = 0
						}
					}
				}

				physicsBody.Body.SetVelocity(dir*ai.MoveSpeed, physicsBody.Body.Velocity().Y)

				return tengo.TrueValue, nil
			}}

			values["move_towards_player_2d"] = &tengo.UserFunction{Name: "move_towards_player_2d", Value: func(args ...tengo.Object) (tengo.Object, error) {
				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				playerTransform, ok := ecs.Get(world, playerEnt, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player Transform component not found")
				}

				dx := playerTransform.X - transform.X
				dy := playerTransform.Y - transform.Y
				dist := math.Hypot(dx, dy)
				if dist < 1e-4 {
					physicsBody.Body.SetVelocity(0, 0)
					return tengo.TrueValue, nil
				}

				stopDistance := ai.AttackRange + 20

				if dist <= stopDistance {
					physicsBody.Body.SetVelocity(0, 0)
					return tengo.TrueValue, nil
				}

				nx := dx / dist
				ny := dy / dist
				physicsBody.Body.SetVelocity(nx*ai.MoveSpeed, ny*ai.MoveSpeed)

				return tengo.TrueValue, nil
			}}

			values["jump_towards_player"] = &tengo.UserFunction{Name: "jump_towards_player", Value: func(args ...tengo.Object) (tengo.Object, error) {
				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				playerPhysicsBody, ok := ecs.Get(world, playerEnt, component.PhysicsBodyComponent.Kind())
				if !ok || playerPhysicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("Player PhysicsBody component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				px, py := playerPhysicsBody.Body.Position().X, playerPhysicsBody.Body.Position().Y
				ex, ey := physicsBody.Body.Position().X, physicsBody.Body.Position().Y

				xOffset := 10.0
				if px < ex {
					xOffset = -10.0
				}

				t := 40.0

				vx := (px - ex + xOffset) / t
				vy := ((py - ey) - 0.5*common.Gravity*math.Pow(t, 2)) / t

				physicsBody.Body.SetVelocity(vx, vy)

				return tengo.TrueValue, nil
			}}

			values["in_attack_range"] = &tengo.UserFunction{Name: "in_attack_range", Value: func(args ...tengo.Object) (tengo.Object, error) {
				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				playerTransform, ok := ecs.Get(world, playerEnt, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player Transform component not found")
				}

				dx := playerTransform.X - transform.X
				dy := playerTransform.Y - transform.Y
				if math.Hypot(dx, dy) > ai.AttackRange+24 {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			values["face_player"] = &tengo.UserFunction{Name: "face_player", Value: func(args ...tengo.Object) (tengo.Object, error) {
				sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Sprite component not found")
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				playerTransform, ok := ecs.Get(world, playerEnt, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player Transform component not found")
				}

				sprite.FacingLeft = playerTransform.X < transform.X

				return tengo.TrueValue, nil
			}}

			values["ground_ahead"] = &tengo.UserFunction{Name: "ground_ahead", Value: func(args ...tengo.Object) (tengo.Object, error) {
				nav, ok := ecs.Get(world, target, component.AINavigationComponent.Kind())
				if !ok || nav == nil {
					return tengo.FalseValue, fmt.Errorf("AI navigation component not found")
				}

				if entityFacingLeft(world, target) {
					return boolObject(nav.GroundAheadLeft), nil
				}

				return boolObject(nav.GroundAheadRight), nil
			}}

			values["forward_blocked"] = &tengo.UserFunction{Name: "forward_blocked", Value: func(args ...tengo.Object) (tengo.Object, error) {
				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || transform == nil {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody == nil || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found")
				}

				probeDistance := 8.0
				if len(args) >= 1 {
					probeDistance = objectAsFloat(args[0])
				}
				if probeDistance < 0 {
					return tengo.FalseValue, fmt.Errorf("probe distance must be non-negative")
				}

				rotation := scriptRotationRadians(world, target, transform)
				forwardX, forwardY := scriptForwardVector(rotation)
				if entityFacingLeft(world, target) {
					forwardX *= -1
					forwardY *= -1
				}

				minX, minY, maxX, maxY := bodyAABB(world, target, transform, physicsBody)
				centerX := (minX + maxX) / 2
				centerY := (minY + maxY) / 2
				faceDistance := math.Abs(forwardX)*(maxX-minX)/2 + math.Abs(forwardY)*(maxY-minY)/2
				if faceDistance <= 0 {
					faceDistance = 16
				}

				_, _, hasHit, _ := firstStaticHit(
					world,
					target,
					centerX,
					centerY,
					centerX+forwardX*(faceDistance+probeDistance),
					centerY+forwardY*(faceDistance+probeDistance),
				)

				return boolObject(hasHit), nil
			}}

			values["lost_player"] = &tengo.UserFunction{Name: "lost_player", Value: func(args ...tengo.Object) (tengo.Object, error) {
				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				playerTransform, ok := ecs.Get(world, playerEnt, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player Transform component not found")
				}

				dx := playerTransform.X - transform.X
				dy := playerTransform.Y - transform.Y

				if math.Hypot(dx, dy) <= ai.FollowRange {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			values["out_attack_range"] = &tengo.UserFunction{Name: "out_attack_range", Value: func(args ...tengo.Object) (tengo.Object, error) {
				ai, ok := ecs.Get(world, target, component.AIComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("AI component not found")
				}

				transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Transform component not found")
				}

				playerEnt, ok := ecs.First(world, component.PlayerComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player not found")
				}

				playerTransform, ok := ecs.Get(world, playerEnt, component.TransformComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Player Transform component not found")
				}

				dx := playerTransform.X - transform.X
				dy := playerTransform.Y - transform.Y
				if math.Hypot(dx, dy) <= ai.AttackRange+34 {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func firstStaticHit(w *ecs.World, player ecs.Entity, x0, y0, x1, y1 float64) (float64, float64, bool, bool) {
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
		if e == player || !body.Static {
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
		if e == player {
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

func scriptEntityPosition(world *ecs.World, entity ecs.Entity) (float64, float64, error) {
	if physicsBody, ok := ecs.Get(world, entity, component.PhysicsBodyComponent.Kind()); ok && physicsBody != nil && physicsBody.Body != nil {
		pos := physicsBody.Body.Position()
		return pos.X, pos.Y, nil
	}

	transform, ok := ecs.Get(world, entity, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return 0, 0, fmt.Errorf("Transform component not found")
	}

	return transform.X, transform.Y, nil
}

type hazardAABB struct {
	x float64
	y float64
	w float64
	h float64
}

func hazardBounds(w *ecs.World, e ecs.Entity, h *component.Hazard, t *component.Transform) (hazardAABB, bool) {
	if h == nil || t == nil || h.Width <= 0 || h.Height <= 0 {
		return hazardAABB{}, false
	}

	// Default: treat transform (t.X,t.Y) as the sprite transform point and
	// interpret hazard offsets relative to that point. Prefer to align the
	// hazard top-left to the sprite's rendered top-left when a Sprite
	// component is present.
	x := aabbTopLeftX(w, e, t.X, h.OffsetX, h.Width, false)
	y := aabbTopLeftY(t.Y, h.OffsetY, h.Height, false)
	wid := h.Width
	hgt := h.Height

	if s, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && s != nil && s.Image != nil {
		// Determine sprite source size
		imgW := s.Image.Bounds().Dx()
		imgH := s.Image.Bounds().Dy()
		if s.UseSource {
			imgW = s.Source.Dx()
			imgH = s.Source.Dy()
		}

		// scaled origin
		originX := s.OriginX * t.ScaleX
		originY := s.OriginY * t.ScaleY

		x = t.X - originX + facingAdjustedOffsetX(w, e, h.OffsetX, wid, false) - wid/2
		y = t.Y - originY + h.OffsetY - hgt/2

		// If spec provided different hazard size, keep it; otherwise use sprite pixel size
		if wid <= 0 {
			wid = float64(imgW) * t.ScaleX
		}
		if hgt <= 0 {
			hgt = float64(imgH) * t.ScaleY
		}
	}

	// If there's no rotation, return the simple AABB.
	if t.Rotation == 0 {
		return hazardAABB{x: x, y: y, w: wid, h: hgt}, true
	}

	// Rotate the four corners of the hazard rect around the transform origin
	// (t.X, t.Y) and compute the axis-aligned bounding box that contains
	// the rotated rectangle. This ensures the collider covers the rotated
	// sprite area for hazard checks.
	cx := t.X
	cy := t.Y
	cosR := math.Cos(t.Rotation)
	sinR := math.Sin(t.Rotation)

	corners := [4][2]float64{
		{x, y},
		{x + wid, y},
		{x, y + hgt},
		{x + wid, y + hgt},
	}

	minX := math.Inf(1)
	minY := math.Inf(1)
	maxX := math.Inf(-1)
	maxY := math.Inf(-1)
	for _, c := range corners {
		dx := c[0] - cx
		dy := c[1] - cy
		rx := dx*cosR - dy*sinR + cx
		ry := dx*sinR + dy*cosR + cy
		if rx < minX {
			minX = rx
		}
		if ry < minY {
			minY = ry
		}
		if rx > maxX {
			maxX = rx
		}
		if ry > maxY {
			maxY = ry
		}
	}

	return hazardAABB{x: minX, y: minY, w: maxX - minX, h: maxY - minY}, true
}

func bodyAABB(w *ecs.World, e ecs.Entity, transform *component.Transform, body *component.PhysicsBody) (minX, minY, maxX, maxY float64) {
	width := body.Width
	height := body.Height
	if width <= 0 {
		width = 32
	}
	if height <= 0 {
		height = 32
	}
	minX = aabbTopLeftX(w, e, transform.X, body.OffsetX, width, body.AlignTopLeft)
	minY = aabbTopLeftY(transform.Y, body.OffsetY, height, body.AlignTopLeft)
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

func segmentCircleHit(w *ecs.World, e ecs.Entity, x0, y0, x1, y1 float64, transform *component.Transform, body *component.PhysicsBody) (float64, float64, bool) {
	r := body.Radius
	if r <= 0 {
		return 0, 0, false
	}

	centerX := bodyCenterX(w, e, transform, &component.PhysicsBody{OffsetX: body.OffsetX, OffsetY: body.OffsetY, Width: 2 * r, Height: 2 * r, AlignTopLeft: body.AlignTopLeft})
	centerY := aabbTopLeftY(transform.Y, body.OffsetY, 2*r, body.AlignTopLeft) + r

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

func entityFacingLeft(w *ecs.World, e ecs.Entity) bool {
	if w == nil || !e.Valid() {
		return false
	}
	s, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
	return ok && s != nil && s.FacingLeft
}

func entitySpriteWidth(w *ecs.World, e ecs.Entity) (float64, bool) {
	if w == nil || !e.Valid() {
		return 0, false
	}
	s, ok := ecs.Get(w, e, component.SpriteComponent.Kind())
	if !ok || s == nil || s.Image == nil {
		return 0, false
	}
	if s.UseSource {
		srcW := s.Source.Dx()
		if srcW > 0 {
			return float64(srcW), true
		}
	}
	wid := s.Image.Bounds().Dx()
	if wid <= 0 {
		return 0, false
	}
	return float64(wid), true
}

func facingAdjustedOffsetX(w *ecs.World, e ecs.Entity, offsetX, aabbWidth float64, alignTopLeft bool) float64 {
	if !entityFacingLeft(w, e) {
		return offsetX
	}
	if spriteW, ok := entitySpriteWidth(w, e); ok && spriteW > 0 {
		if alignTopLeft {
			return spriteW - offsetX - aabbWidth
		}
		return spriteW - offsetX
	}
	if alignTopLeft {
		return -offsetX - aabbWidth
	}
	return -offsetX
}

func aabbTopLeftX(w *ecs.World, e ecs.Entity, transformX, offsetX, aabbWidth float64, alignTopLeft bool) float64 {
	effectiveOffsetX := facingAdjustedOffsetX(w, e, offsetX, aabbWidth, alignTopLeft)
	if alignTopLeft {
		return transformX + effectiveOffsetX
	}
	return transformX + effectiveOffsetX - aabbWidth/2
}

func aabbTopLeftY(transformY, offsetY, aabbHeight float64, alignTopLeft bool) float64 {
	if alignTopLeft {
		return transformY + offsetY
	}
	return transformY + offsetY - aabbHeight/2
}

func bodyCenterX(w *ecs.World, e ecs.Entity, t *component.Transform, body *component.PhysicsBody) float64 {
	if t == nil || body == nil {
		return 0
	}
	effectiveOffsetX := facingAdjustedOffsetX(w, e, body.OffsetX, body.Width, body.AlignTopLeft)
	centerX := t.X + effectiveOffsetX
	if body.AlignTopLeft {
		centerX += body.Width / 2
	}
	return centerX
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
