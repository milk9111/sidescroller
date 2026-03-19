package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func BreakableWallModule() Module {
	return Module{
		Name: "breakable_wall",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["layer_name"] = &tengo.UserFunction{Name: "layer_name", Value: func(args ...tengo.Object) (tengo.Object, error) {
				wall, ok := ecs.Get(world, target, component.BreakableWallComponent.Kind())
				if !ok || wall == nil {
					return tengo.UndefinedValue, fmt.Errorf("breakable wall component not found for entity %v", target)
				}

				return &tengo.String{Value: strings.TrimSpace(wall.LayerName)}, nil
			}}

			return values
		},
	}
}
