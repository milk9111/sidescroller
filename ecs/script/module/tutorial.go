package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
)

func TutorialModule() Module {
	return Module{
		Name: "tutorial",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, _ ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["show"] = &tengo.UserFunction{Name: "show", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("show requires 2 arguments: message and duration frames")
				}

				message := strings.TrimSpace(objectAsString(args[0]))
				frames := objectAsInt(args[1])
				if message == "" {
					return tengo.FalseValue, fmt.Errorf("message must not be empty")
				}

				if err := levelentity.ShowTutorial(world, message, frames); err != nil {
					return tengo.FalseValue, err
				}

				return tengo.TrueValue, nil
			}}

			values["hide"] = &tengo.UserFunction{Name: "hide", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if err := levelentity.HideTutorial(world); err != nil {
					return tengo.FalseValue, err
				}
				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
