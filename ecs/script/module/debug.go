package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
)

func DebugModule() Module {
	return Module{
		Name: "debug",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, _ ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["print"] = &tengo.UserFunction{Name: "print", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 3 {
					return tengo.FalseValue, fmt.Errorf("print requires at least 3 arguments: width, height, and message")
				}

				width := objectAsInt(args[0])
				height := objectAsInt(args[1])
				message := strings.TrimSpace(objectAsString(args[2]))

				if width <= 0 {
					return tengo.FalseValue, fmt.Errorf("width must be a positive integer")
				}

				if height <= 0 {
					return tengo.FalseValue, fmt.Errorf("height must be a positive integer")
				}

				if message == "" {
					return tengo.FalseValue, fmt.Errorf("message must not be empty")
				}

				if err := levelentity.ShowDebugMessage(world, width, height, message); err != nil {
					return tengo.FalseValue, err
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
