package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func SpriteModule() Module {
	return Module{
		Name: "sprite",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: is_facing_left() -> bool
			// doc: Returns true if the sprite is currently facing left.
			// sig: is_facing_left() -> bool
			// doc: Returns true if the sprite is currently facing left.
			values["is_facing_left"] = &tengo.UserFunction{Name: "is_facing_left", Value: func(args ...tengo.Object) (tengo.Object, error) {
				sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Sprite component is required")
				}

				if !sprite.FacingLeft {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			// sig: set_facing_left(value bool) -> bool
			// doc: Sets the sprite facing direction; true = left.
			// sig: set_facing_left(left bool) -> bool
			// doc: Set whether the sprite should face left.
			values["set_facing_left"] = &tengo.UserFunction{Name: "set_facing_left", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("set_facing_left requires 1 argument: boolean value")
				}

				facingLeft := objectAsBool(args[0])
				sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("Sprite component is required")
				}

				sprite.FacingLeft = facingLeft

				return tengo.TrueValue, nil
			}}

			// sig: add_white_flash(duration int) -> bool
			// doc: Adds a white flash effect for `duration` frames.
			// sig: add_white_flash(duration float) -> bool
			// doc: Add a brief white flash effect to the sprite for the given duration.
			values["add_white_flash"] = &tengo.UserFunction{Name: "add_white_flash", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("add_white_flash requires 1 argument: duration in frames")
				}

				duration := objectAsInt(args[0])
				if duration < 0 {
					return tengo.FalseValue, fmt.Errorf("duration must be non-negative")
				}

				err := ecs.Add(world, target, component.WhiteFlashComponent.Kind(), &component.WhiteFlash{Frames: duration, Interval: 5, Timer: 0, On: true})
				if err != nil {
					return tengo.FalseValue, fmt.Errorf("failed to add WhiteFlash component: %v", err)
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func objectAsBool(obj tengo.Object) bool {
	switch v := obj.(type) {
	case *tengo.Bool:
		return !v.IsFalsy()
	case *tengo.String:
		return v.Value == "true"
	default:
		panic("unsupported type for objectAsBool")
	}
}
