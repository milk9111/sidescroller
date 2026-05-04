package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func AnimationModule() Module {
	return Module{
		Name: "animation",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: set(name string) -> bool
			// doc: Set the current animation by name. Returns true when changed.
			values["set"] = &tengo.UserFunction{Name: "set", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("set requires at least 1 argument: the name of the animation to set")
				}

				name := objectAsString(args[0])
				if name == "" {
					return tengo.FalseValue, fmt.Errorf("set requires a valid animation name")
				}

				animation, ok := ecs.Get(world, target, component.AnimationComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("set failed: animation component not found")
				}

				if _, ok := animation.Defs[name]; !ok {
					return tengo.FalseValue, fmt.Errorf("set failed: animation '%s' not found", name)
				}

				animation.Current = name
				animation.Frame = 0
				animation.FrameTimer = 0
				animation.FrameProgress = 0
				animation.Playing = true

				return tengo.TrueValue, nil
			}}

			// sig: finished() -> bool
			// doc: Returns true if the current animation has finished playing.
			values["finished"] = &tengo.UserFunction{Name: "finished", Value: func(args ...tengo.Object) (tengo.Object, error) {
				animation, ok := ecs.Get(world, target, component.AnimationComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("finished failed: animation component not found")
				}

				if animation.Playing || animation.Frame != animation.Defs[animation.Current].FrameCount-1 {
					return tengo.FalseValue, nil
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}
