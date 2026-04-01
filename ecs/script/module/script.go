package module

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func ScriptModule() Module {
	return Module{
		Name: "script",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: add_path(path string) -> bool
			// doc: Appends a script path to the entity and resets script startup so it is rebuilt on the next update.
			values["add_path"] = &tengo.UserFunction{Name: "add_path", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("add_path requires 1 argument: path")
				}

				path := normalizeScriptPath(objectAsString(args[0]))
				if path == "" {
					return tengo.FalseValue, fmt.Errorf("add_path requires a non-empty path")
				}

				if _, err := prefabs.LoadScript(path); err != nil {
					return tengo.FalseValue, fmt.Errorf("add_path failed to load %q: %w", path, err)
				}

				scriptComp, ok := ecs.Get(world, target, component.ScriptComponent.Kind())
				if !ok || scriptComp == nil {
					scriptComp = &component.Script{Paths: []string{path}}
				} else {
					paths := make([]string, 0, len(scriptComp.Paths)+2)
					if len(scriptComp.Paths) > 0 {
						for _, existingPath := range scriptComp.Paths {
							normalized := normalizeScriptPath(existingPath)
							if normalized != "" {
								paths = append(paths, normalized)
							}
						}
					} else if legacyPath := normalizeScriptPath(scriptComp.Path); legacyPath != "" {
						paths = append(paths, legacyPath)
					}

					for _, existingPath := range paths {
						if existingPath == path {
							resetScriptRuntime(world, target)
							return tengo.TrueValue, nil
						}
					}

					scriptComp.Paths = append(paths, path)
				}

				if err := ecs.Add(world, target, component.ScriptComponent.Kind(), scriptComp); err != nil {
					return tengo.FalseValue, fmt.Errorf("add_path failed to update Script component: %w", err)
				}

				resetScriptRuntime(world, target)
				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func normalizeScriptPath(path string) string {
	s := filepath.ToSlash(strings.TrimSpace(path))
	if s == "" {
		return ""
	}

	if after, ok := strings.CutPrefix(s, "prefabs/scripts/"); ok {
		s = after
	}
	if after, ok := strings.CutPrefix(s, "prefabs/"); ok {
		s = after
	}
	if after, ok := strings.CutPrefix(s, "scripts/"); ok {
		s = after
	}

	return s
}

func resetScriptRuntime(world *ecs.World, target ecs.Entity) {
	runtimeComp, ok := ecs.Get(world, target, component.ScriptRuntimeComponent.Kind())
	if !ok || runtimeComp == nil {
		return
	}

	runtimeComp.Started = false
}
