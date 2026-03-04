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
				// TODO - make this an actual disable instead of removing the component entirely, which would lose all state on the hazard (e.g. damage, knockback, etc.)
				if ok := ecs.Remove(world, target, component.HazardComponent.Kind()); !ok {
					return tengo.FalseValue, fmt.Errorf("entity does not have a hazard component")
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
