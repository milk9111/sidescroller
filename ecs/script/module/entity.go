package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func EntityModule() Module {
	return Module{
		Name: "entity",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}
			// sig: id() -> string
			// doc: Returns the game entity id string for this entity.
			// sig: id() -> int
			// doc: Returns the entity's numeric id.
			values["id"] = &tengo.UserFunction{Name: "id", Value: func(args ...tengo.Object) (tengo.Object, error) {
				id, ok := ecs.Get(world, target, component.GameEntityIDComponent.Kind())
				if !ok || id == nil {
					return &tengo.String{Value: ""}, fmt.Errorf("entity does not have a GameEntityID component")
				}

				return &tengo.String{Value: id.Value}, nil
			}}

			// sig: destroy() -> bool
			// doc: Destroys this entity immediately. Returns true if destruction was scheduled/applied.
			values["destroy"] = &tengo.UserFunction{Name: "destroy", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if ecs.DestroyEntity(world, target) {
					return tengo.TrueValue, nil
				}

				return tengo.FalseValue, nil
			}}
			return values
		},
	}
}

func objectAsFloat(obj tengo.Object) float64 {
	switch v := obj.(type) {
	case *tengo.Int:
		return float64(v.Value)
	case *tengo.Float:
		return v.Value
	case *tengo.String:
		var out float64
		_, _ = fmt.Sscanf(v.Value, "%f", &out)
		return out
	default:
		panic("unsupported type for objectAsFloat")
	}
}
