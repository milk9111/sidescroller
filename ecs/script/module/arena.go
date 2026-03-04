package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func ArenaModule() Module {
	return Module{
		Name: "arena",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, _ ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["activate"] = &tengo.UserFunction{Name: "activate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("activate requires 1 argument: group name")
				}

				group := strings.TrimSpace(objectAsString(args[0]))
				if group == "" {
					return tengo.FalseValue, fmt.Errorf("invalid arena group name")
				}

				ecs.ForEach(world, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
					if node == nil || node.Group != group {
						return
					}

					node.Active = true
				})

				return tengo.TrueValue, nil
			}}

			values["deactivate"] = &tengo.UserFunction{Name: "activate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("activate requires 1 argument: group name")
				}

				group := strings.TrimSpace(objectAsString(args[0]))
				if group == "" {
					return tengo.FalseValue, fmt.Errorf("invalid arena group name")
				}

				ecs.ForEach(world, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
					if node == nil || node.Group != group {
						return
					}

					node.Active = false
				})

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
