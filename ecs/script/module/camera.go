package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func CameraModule() Module {
	return Module{
		Name: "camera",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["shake"] = &tengo.UserFunction{Name: "shake", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("shake requires 2 arguments: duration and intensity")
				}

				duration := objectAsInt(args[0])
				intensity := objectAsFloat(args[1])

				if duration < 0 {
					return tengo.FalseValue, fmt.Errorf("duration must be non-negative")
				}

				if intensity < 0 {
					return tengo.FalseValue, fmt.Errorf("intensity must be non-negative")
				}

				camEnt, ok := ecs.First(world, component.CameraTagComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("camera entity not found")
				}

				err := ecs.Add(world, camEnt, component.CameraShakeRequestComponent.Kind(), &component.CameraShakeRequest{
					Frames:    duration,
					Intensity: intensity,
				})
				if err != nil {
					return tengo.FalseValue, fmt.Errorf("failed to add CameraShakeRequest component to camera entity: %w", err)
				}

				return tengo.TrueValue, nil
			}}

			values["lock"] = &tengo.UserFunction{Name: "lock", Value: func(args ...tengo.Object) (tengo.Object, error) {
				camEnt, ok := ecs.First(world, component.CameraTagComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("camera entity not found")
				}

				cam, ok := ecs.Get(world, camEnt, component.CameraComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("camera component not found for camera entity")
				}

				cam.LockCapture = true

				return tengo.TrueValue, nil
			}}

			values["unlock"] = &tengo.UserFunction{Name: "unlock", Value: func(args ...tengo.Object) (tengo.Object, error) {
				camEnt, ok := ecs.First(world, component.CameraTagComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("camera entity not found")
				}

				cam, ok := ecs.Get(world, camEnt, component.CameraComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("camera component not found for camera entity")
				}

				cam.LockCapture = false

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func objectAsInt(obj tengo.Object) int {
	switch v := obj.(type) {
	case *tengo.Int:
		return int(v.Value)
	case *tengo.Float:
		return int(v.Value)
	case *tengo.String:
		var out int
		_, _ = fmt.Sscanf(v.Value, "%d", &out)
		return out
	default:
		panic("unsupported type for objectAsInt")
	}
}
