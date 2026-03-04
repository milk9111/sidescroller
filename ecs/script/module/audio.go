package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func AudioModule() Module {
	return Module{
		Name: "audio",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}
			values["play"] = &tengo.UserFunction{Name: "play", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("play requires at least 1 argument: the name of the audio to play")
				}

				name := strings.TrimSpace(objectAsString(args[0]))
				if name == "" {
					return tengo.FalseValue, fmt.Errorf("invalid audio name")
				}

				audio, ok := ecs.Get(world, target, component.AudioComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("audio component not found")
				}

				for i, clipName := range audio.Names {
					if clipName == name {
						audio.Play[i] = true
						return tengo.TrueValue, nil
					}
				}

				return tengo.FalseValue, fmt.Errorf("audio clip not found")
			}}

			values["stop"] = &tengo.UserFunction{Name: "stop", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("stop requires at least 1 argument: the name of the audio to stop")
				}

				name := strings.TrimSpace(objectAsString(args[0]))
				if name == "" {
					return tengo.FalseValue, fmt.Errorf("invalid audio name")
				}

				audio, ok := ecs.Get(world, target, component.AudioComponent.Kind())
				if !ok {
					return tengo.FalseValue, fmt.Errorf("audio component not found")
				}

				for i, audioName := range audio.Names {
					if audioName != name {
						continue
					}

					audio.Stop[i] = true
					break
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func objectAsString(obj tengo.Object) string {
	switch v := obj.(type) {
	case *tengo.String:
		return v.Value
	case *tengo.Int:
		return fmt.Sprintf("%d", v.Value)
	case *tengo.Float:
		return fmt.Sprintf("%f", v.Value)
	default:
		return ""
	}
}
