package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func LineRenderModule() Module {
	return Module{
		Name: "line_render",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["set_points"] = &tengo.UserFunction{Name: "set_points", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 4 {
					return tengo.FalseValue, fmt.Errorf("set_points requires 4 arguments: start_x, start_y, end_x, end_y")
				}

				line, ok := ecs.Get(world, target, component.LineRenderComponent.Kind())
				if !ok || line == nil {
					return tengo.FalseValue, fmt.Errorf("LineRender component is required")
				}

				line.StartX = objectAsFloat(args[0])
				line.StartY = objectAsFloat(args[1])
				line.EndX = objectAsFloat(args[2])
				line.EndY = objectAsFloat(args[3])

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
