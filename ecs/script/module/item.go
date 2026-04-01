package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func ItemModule() Module {
	return Module{
		Name: "item",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: set_item_reference(path string) -> bool
			// doc: Sets the ItemReference prefab path on the target entity.
			values["set_item_reference"] = &tengo.UserFunction{Name: "set_item_reference", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("set_item_reference requires 1 argument: path")
				}

				prefabPath := strings.TrimSpace(objectAsString(args[0]))
				if prefabPath == "" {
					return tengo.FalseValue, fmt.Errorf("set_item_reference requires a non-empty path")
				}

				itemReference, ok := ecs.Get(world, target, component.ItemReferenceComponent.Kind())
				if !ok || itemReference == nil {
					return tengo.FalseValue, fmt.Errorf("ItemReference component not found for entity %v", target)
				}

				itemReference.Prefab = prefabPath
				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
