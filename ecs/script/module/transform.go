package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TransformModule() Module {
	return Module{
		Name: "transform",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}
			// sig: position() -> (float, float)
			// doc: Returns the current position as an [x, y] array of floats.
			// sig: position() -> map
			// doc: Returns a map with `x` and `y` numeric fields for the entity's position.
			values["position"] = &tengo.UserFunction{Name: "position", Value: func(args ...tengo.Object) (tengo.Object, error) {
				tf, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || tf == nil {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, fmt.Errorf("entity does not have a transform component")
				}

				return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: tf.X}, &tengo.Float{Value: tf.Y}}}, nil
			}}
			// sig: set_position(x float, y float) -> bool
			// doc: Sets the entity transform position to (x, y).
			// sig: set_position(x float, y float) -> bool
			// doc: Set the entity's position to the given x,y coordinates.
			values["set_position"] = &tengo.UserFunction{Name: "set_position", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("set_position requires 2 arguments: x and y")
				}

				x := objectAsFloat(args[0])
				y := objectAsFloat(args[1])

				tf, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || tf == nil {
					return tengo.FalseValue, fmt.Errorf("entity does not have a transform component")
				}

				tf.X = x
				tf.Y = y

				if tf.ScaleX == 0 {
					tf.ScaleX = 1
				}

				if tf.ScaleY == 0 {
					tf.ScaleY = 1
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
