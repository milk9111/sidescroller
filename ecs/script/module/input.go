package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func InputModule() Module {
	return Module{
		Name: "input",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["stop"] = &tengo.UserFunction{Name: "stop", Value: func(args ...tengo.Object) (tengo.Object, error) {
				p, ok := ecs.First(world, component.PlayerTagComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("could not find player entity")
				}

				input, ok := ecs.Get(world, p, component.InputComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("input component not found for player entity")
				}

				input.Disabled = true

				return tengo.TrueValue, nil
			}}

			values["start"] = &tengo.UserFunction{Name: "start", Value: func(args ...tengo.Object) (tengo.Object, error) {
				p, ok := ecs.First(world, component.PlayerTagComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("could not find player entity")
				}

				input, ok := ecs.Get(world, p, component.InputComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("input component not found for player entity")
				}

				input.Disabled = false

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
