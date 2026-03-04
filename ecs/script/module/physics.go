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

			values["stop_x"] = &tengo.UserFunction{Name: "stop_x", Value: func(args ...tengo.Object) (tengo.Object, error) {
				physicsBody, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind())
				if !ok || physicsBody.Body == nil {
					return tengo.FalseValue, fmt.Errorf("PhysicsBody component not found for entity %v", target)
				}

				physicsBody.Body.SetVelocity(0, physicsBody.Body.Velocity().Y)
				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
