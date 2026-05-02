package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func HealthModule() Module {
	return Module{
		Name: "health",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: initial() -> int
			// doc: Returns the initial health value for the entity.
			values["initial"] = &tengo.UserFunction{Name: "initial", Value: func(args ...tengo.Object) (tengo.Object, error) {
				health, ok := ecs.Get(world, target, component.HealthComponent.Kind())
				if !ok {
					return &tengo.Int{Value: 0}, nil
				}

				return &tengo.Int{Value: int64(health.Initial)}, nil
			}}

			// sig: current() -> int
			// doc: Returns the current health value for the entity.
			// sig: current() -> int
			// doc: Returns the current health value as an integer.
			values["current"] = &tengo.UserFunction{Name: "current", Value: func(args ...tengo.Object) (tengo.Object, error) {
				health, ok := ecs.Get(world, target, component.HealthComponent.Kind())
				if !ok {
					return &tengo.Int{Value: 0}, nil
				}

				return &tengo.Int{Value: int64(health.Current)}, nil
			}}

			// sig: invulnerability_activate(frames? int) -> bool
			// doc: Adds invulnerability to the entity. Optional frames argument (0 means indefinite).
			values["invulnerability_activate"] = &tengo.UserFunction{Name: "invulnerability_activate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				frames := 0
				if len(args) > 0 {
					frames = int(objectAsFloat(args[0]))
					if frames < 0 {
						return tengo.FalseValue, fmt.Errorf("invulnerability_activate failed: frames argument must be non-negative")
					}
				}

				if err := ecs.Add(world, target, component.InvulnerableComponent.Kind(), &component.Invulnerable{Frames: frames}); err != nil {
					return tengo.FalseValue, fmt.Errorf("invulnerability_activate failed: %v", err)
				}

				return tengo.TrueValue, nil
			}}

			// sig: invulnerability_deactivate() -> bool
			// doc: Removes invulnerability from the entity.
			values["invulnerability_deactivate"] = &tengo.UserFunction{Name: "invulnerability_deactivate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if ecs.Remove(world, target, component.InvulnerableComponent.Kind()) {
					return tengo.TrueValue, nil
				}

				return tengo.FalseValue, fmt.Errorf("invulnerability_deactivate failed: entity was not invulnerable")
			}}

			return values
		},
	}
}
