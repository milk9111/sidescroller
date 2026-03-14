package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func HazardModule() Module {
	return Module{
		Name: "hazard",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["disable"] = &tengo.UserFunction{Name: "disable", Value: func(args ...tengo.Object) (tengo.Object, error) {
				hazard, ok := ecs.Get(world, target, component.HazardComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("hazard component not found for entity %v", target)
				}

				hazard.Disabled = true

				return tengo.TrueValue, nil
			}}

			values["enable"] = &tengo.UserFunction{Name: "enable", Value: func(args ...tengo.Object) (tengo.Object, error) {
				hazard, ok := ecs.Get(world, target, component.HazardComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("hazard component not found for entity %v", target)
				}

				hazard.Disabled = false

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
