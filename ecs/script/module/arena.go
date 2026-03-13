package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func ArenaModule() Module {
	return Module{
		Name: "arena",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, owner ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["is_active"] = &tengo.UserFunction{Name: "is_active", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) == 0 {
					node, ok := ecs.Get(world, target, component.ArenaNodeComponent.Kind())
					if !ok || node == nil || !node.Active {
						return tengo.FalseValue, nil
					}
					return tengo.TrueValue, nil
				}

				group := strings.TrimSpace(objectAsString(args[0]))
				if group == "" {
					return tengo.FalseValue, fmt.Errorf("invalid arena group name")
				}

				active := false
				ecs.ForEach(world, component.ArenaNodeComponent.Kind(), func(_ ecs.Entity, node *component.ArenaNode) {
					if node != nil && node.Group == group && node.Active {
						active = true
					}
				})

				if active {
					return tengo.TrueValue, nil
				}
				return tengo.FalseValue, nil
			}}

			values["activate"] = &tengo.UserFunction{Name: "activate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("activate requires 1 argument: group name")
				}

				group := strings.TrimSpace(objectAsString(args[0]))
				if group == "" {
					return tengo.FalseValue, fmt.Errorf("invalid arena group name")
				}

				ecs.ForEach(world, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
					if node == nil || node.Group != group {
						return
					}

					node.Active = true
					emitArenaSignal(world, ent, owner, "activate")
				})

				return tengo.TrueValue, nil
			}}

			values["deactivate"] = &tengo.UserFunction{Name: "activate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("activate requires 1 argument: group name")
				}

				group := strings.TrimSpace(objectAsString(args[0]))
				if group == "" {
					return tengo.FalseValue, fmt.Errorf("invalid arena group name")
				}

				ecs.ForEach(world, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
					if node == nil || node.Group != group {
						return
					}

					node.Active = false
					emitArenaSignal(world, ent, owner, "deactivate")
				})

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func emitArenaSignal(world *ecs.World, target ecs.Entity, source ecs.Entity, signalName string) {
	if world == nil || !target.Valid() {
		return
	}

	sourceGameEntity := ""
	if source.Valid() && ecs.IsAlive(world, source) {
		if id, ok := ecs.Get(world, source, component.GameEntityIDComponent.Kind()); ok && id != nil {
			sourceGameEntity = strings.TrimSpace(id.Value)
		}
	}

	queue, ok := ecs.Get(world, target, component.ScriptSignalQueueComponent.Kind())
	if !ok || queue == nil {
		queue = &component.ScriptSignalQueue{}
	}
	queue.Events = append(queue.Events, component.ScriptSignalEvent{Name: signalName, SourceGameEntity: sourceGameEntity})
	_ = ecs.Add(world, target, component.ScriptSignalQueueComponent.Kind(), queue)
}
