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

			values["disable"] = &tengo.UserFunction{Name: "disable", Value: func(args ...tengo.Object) (tengo.Object, error) {
				sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
				if !ok || sprite == nil {
					return tengo.FalseValue, fmt.Errorf("Sprite component is required")
				}

				sprite.Disabled = true

				return tengo.TrueValue, nil
			}}

			values["enable"] = &tengo.UserFunction{Name: "enable", Value: func(args ...tengo.Object) (tengo.Object, error) {
				sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
				if !ok || sprite == nil {
					return tengo.FalseValue, fmt.Errorf("Sprite component is required")
				}

				sprite.Disabled = false

				return tengo.TrueValue, nil
			}}

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

			// sig: add_shake(duration int, intensity float) -> bool
			// doc: Adds a temporary random shake offset to the sprite.
			values["add_shake"] = &tengo.UserFunction{Name: "add_shake", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("add_shake requires 2 arguments: duration in frames and intensity")
				}

				duration := objectAsInt(args[0])
				intensity := objectAsFloat(args[1])
				if duration < 0 {
					return tengo.FalseValue, fmt.Errorf("duration must be non-negative")
				}
				if intensity < 0 {
					return tengo.FalseValue, fmt.Errorf("intensity must be non-negative")
				}

				shake, ok := ecs.Get(world, target, component.SpriteShakeComponent.Kind())
				if !ok || shake == nil {
					shake = &component.SpriteShake{}
				}
				if duration > shake.Frames {
					shake.Frames = duration
				}
				if intensity > shake.Intensity {
					shake.Intensity = intensity
				}

				if err := ecs.Add(world, target, component.SpriteShakeComponent.Kind(), shake); err != nil {
					return tengo.FalseValue, fmt.Errorf("failed to add SpriteShake component: %v", err)
				}

				return tengo.TrueValue, nil
			}}

			// sig: add_fade_out(duration int) -> bool
			// doc: Fades the sprite out over `duration` frames.
			values["add_fade_out"] = &tengo.UserFunction{Name: "add_fade_out", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("add_fade_out requires 1 argument: duration in frames")
				}

				duration := objectAsInt(args[0])
				if duration < 0 {
					return tengo.FalseValue, fmt.Errorf("duration must be non-negative")
				}

				if err := ecs.Add(world, target, component.SpriteFadeOutComponent.Kind(), &component.SpriteFadeOut{
					Frames:      duration,
					TotalFrames: duration,
					Alpha:       1,
				}); err != nil {
					return tengo.FalseValue, fmt.Errorf("failed to add SpriteFadeOut component: %v", err)
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
