package module

import (
	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func HealthModule() Module {
	return Module{
		Name: "health",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["current"] = &tengo.UserFunction{Name: "current", Value: func(args ...tengo.Object) (tengo.Object, error) {
				health, ok := ecs.Get(world, target, component.HealthComponent.Kind())
				if !ok {
					return &tengo.Int{Value: 0}, nil
				}

				return &tengo.Int{Value: int64(health.Current)}, nil
			}}

			return values
		},
	}
}
