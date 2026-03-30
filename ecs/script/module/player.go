package module

import (
	"fmt"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func PlayerModule() Module {
	return Module{
		Name: "player",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, _ ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["add_gear"] = &tengo.UserFunction{Name: "add_gear", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return &tengo.Int{Value: 0}, fmt.Errorf("add_gear requires 1 argument: amount")
				}

				amount := objectAsInt(args[0])
				gears := ensurePlayerGearCountComponent(world)
				if gears == nil {
					return &tengo.Int{Value: 0}, nil
				}

				gears.Count += amount
				return &tengo.Int{Value: int64(gears.Count)}, nil
			}}

			values["gear_count"] = &tengo.UserFunction{Name: "gear_count", Value: func(args ...tengo.Object) (tengo.Object, error) {
				gears := ensurePlayerGearCountComponent(world)
				if gears == nil {
					return &tengo.Int{Value: 0}, nil
				}
				return &tengo.Int{Value: int64(gears.Count)}, nil
			}}

			return values
		},
	}
}

func ensurePlayerGearCountComponent(world *ecs.World) *component.PlayerGearCount {
	if world == nil {
		return nil
	}

	if ent, ok := ecs.First(world, component.PlayerGearCountComponent.Kind()); ok {
		if gears, ok := ecs.Get(world, ent, component.PlayerGearCountComponent.Kind()); ok && gears != nil {
			return gears
		}
	}

	ent := ecs.CreateEntity(world)
	gears := &component.PlayerGearCount{}
	_ = ecs.Add(world, ent, component.PersistentComponent.Kind(), &component.Persistent{
		ID:                "player_gears",
		KeepOnLevelChange: true,
		KeepOnReload:      false,
	})
	_ = ecs.Add(world, ent, component.PlayerGearCountComponent.Kind(), gears)
	return gears
}
