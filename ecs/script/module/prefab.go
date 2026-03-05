package module

import (
	"fmt"
	"time"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	entitypkg "github.com/milk9111/sidescroller/ecs/entity"
)

func PrefabModule() Module {
	return Module{
		Name: "prefab",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, _ ecs.Entity, _ ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: instantiate(path string) -> string
			// doc: Instantiates a prefab by path and returns the new game entity id, or empty string on failure.
			// sig: instantiate(name string, x float, y float) -> int
			// doc: Instantiate a prefab by name at the given coordinates; returns the new entity id.
			values["instantiate"] = &tengo.UserFunction{Name: "instantiate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if world == nil {
					return &tengo.String{Value: ""}, fmt.Errorf("prefab.instantiate: world is nil")
				}
				if len(args) < 1 {
					return &tengo.String{Value: ""}, nil
				}

				prefabPath := ""
				if s, ok := args[0].(*tengo.String); ok {
					prefabPath = s.Value
				} else {
					prefabPath = args[0].String()
				}
				if prefabPath == "" {
					return &tengo.String{Value: ""}, nil
				}

				ent, err := entitypkg.BuildEntity(world, prefabPath)
				if err != nil {
					return &tengo.String{Value: ""}, fmt.Errorf("prefab.instantiate: build %q: %w", prefabPath, err)
				}

				// generate a reasonably-unique game entity id
				id := fmt.Sprintf("p%d", time.Now().UnixNano())
				if byGameEntityID != nil {
					for {
						if _, exists := byGameEntityID[id]; !exists {
							break
						}
						id = fmt.Sprintf("p%d", time.Now().UnixNano())
					}
				}

				if err := ecs.Add(world, ent, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: id}); err != nil {
					return &tengo.String{Value: ""}, fmt.Errorf("prefab.instantiate: add GameEntityID: %w", err)
				}

				byGameEntityID[id] = ent

				return &tengo.String{Value: id}, nil
			}}

			return values
		},
	}
}
