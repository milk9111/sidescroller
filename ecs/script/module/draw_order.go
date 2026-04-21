package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func DrawOrderModule() Module {
	return Module{
		Name: "draw_order",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["copy_behind_owner"] = &tengo.UserFunction{Name: "copy_behind_owner", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if !owner.Valid() || !ecs.IsAlive(world, owner) {
					return tengo.FalseValue, fmt.Errorf("owner entity is not alive")
				}
				if err := copyDrawOrder(world, owner, target, -1); err != nil {
					return tengo.FalseValue, err
				}
				return tengo.TrueValue, nil
			}}

			values["copy_behind"] = &tengo.UserFunction{Name: "copy_behind", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("copy_behind requires 1 argument: source entity id")
				}

				sourceID, ok := args[0].(*tengo.String)
				if !ok || sourceID.Value == "" {
					return tengo.FalseValue, fmt.Errorf("copy_behind requires a non-empty entity id")
				}

				source, ok := byGameEntityID[sourceID.Value]
				if !ok || !source.Valid() || !ecs.IsAlive(world, source) {
					return tengo.FalseValue, fmt.Errorf("source entity %q is not alive", sourceID.Value)
				}

				if err := copyDrawOrder(world, source, target, -1); err != nil {
					return tengo.FalseValue, err
				}

				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func copyDrawOrder(world *ecs.World, source, target ecs.Entity, renderOffset int) error {
	if world == nil {
		return fmt.Errorf("world is nil")
	}
	if !target.Valid() || !ecs.IsAlive(world, target) {
		return fmt.Errorf("target entity is not alive")
	}

	if sourceLayer, ok := ecs.Get(world, source, component.EntityLayerComponent.Kind()); ok && sourceLayer != nil {
		copiedLayer := *sourceLayer
		if err := ecs.Add(world, target, component.EntityLayerComponent.Kind(), &copiedLayer); err != nil {
			return fmt.Errorf("copy entity layer: %w", err)
		}
	}

	renderIndex := renderOffset
	if sourceRender, ok := ecs.Get(world, source, component.RenderLayerComponent.Kind()); ok && sourceRender != nil {
		renderIndex = sourceRender.Index + renderOffset
	}

	if err := ecs.Add(world, target, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: renderIndex}); err != nil {
		return fmt.Errorf("copy render layer: %w", err)
	}

	return nil
}
