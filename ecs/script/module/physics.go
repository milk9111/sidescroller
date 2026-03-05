package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func PhysicsModule() Module {
	return Module{
		Name: "physics",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: stop_x() -> bool
			// doc: Stops horizontal movement on the physics body.
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

			// sig: jump(velocity float) -> bool
			// doc: Applies a vertical velocity (jump) to the physics body.
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
			// doc: Returns true if the entity is currently grounded.
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
