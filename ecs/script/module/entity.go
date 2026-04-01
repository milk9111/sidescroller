package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func EntityModule() Module {
	return Module{
		Name: "entity",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			// sig: id() -> int
			// doc: Returns the entity's numeric id.
			values["id"] = &tengo.UserFunction{Name: "id", Value: func(args ...tengo.Object) (tengo.Object, error) {
				id, ok := ecs.Get(world, target, component.GameEntityIDComponent.Kind())
				if !ok || id == nil {
					return &tengo.String{Value: ""}, fmt.Errorf("entity does not have a GameEntityID component")
				}

				return &tengo.String{Value: id.Value}, nil
			}}

			// sig: destroy() -> bool
			// doc: Destroys this entity immediately. Returns true if destruction was scheduled/applied.
			values["destroy"] = &tengo.UserFunction{Name: "destroy", Value: func(args ...tengo.Object) (tengo.Object, error) {
				recordDestroyedLevelEntityState(world, target)
				if ecs.DestroyEntity(world, target) {
					return tengo.TrueValue, nil
				}

				return tengo.FalseValue, nil
			}}
			return values
		},
	}
}

func recordDestroyedLevelEntityState(world *ecs.World, target ecs.Entity) {
	if world == nil || !target.Valid() || !ecs.IsAlive(world, target) {
		return
	}
	if ecs.Has(world, target, component.PlayerTagComponent.Kind()) {
		return
	}

	levelName := currentLevelNameForScript(world)
	if levelName == "" {
		return
	}

	gameID, ok := ecs.Get(world, target, component.GameEntityIDComponent.Kind())
	if !ok || gameID == nil || strings.TrimSpace(gameID.Value) == "" {
		return
	}

	player, ok := ecs.First(world, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	stateMap, ok := ecs.Get(world, player, component.LevelEntityStateMapComponent.Kind())
	if !ok || stateMap == nil {
		stateMap = &component.LevelEntityStateMap{States: map[string]component.PersistedLevelEntityState{}}
		_ = ecs.Add(world, player, component.LevelEntityStateMapComponent.Kind(), stateMap)
	}
	if stateMap.States == nil {
		stateMap.States = map[string]component.PersistedLevelEntityState{}
	}

	stateMap.States[levelName+"#"+gameID.Value] = component.PersistedLevelEntityStateDefeated
}

func currentLevelNameForScript(world *ecs.World) string {
	if world == nil {
		return ""
	}

	ent, ok := ecs.First(world, component.LevelRuntimeComponent.Kind())
	if !ok {
		return ""
	}

	runtimeComp, ok := ecs.Get(world, ent, component.LevelRuntimeComponent.Kind())
	if !ok || runtimeComp == nil {
		return ""
	}

	return strings.TrimSpace(runtimeComp.Name)
}

func objectAsFloat(obj tengo.Object) float64 {
	switch v := obj.(type) {
	case *tengo.Int:
		return float64(v.Value)
	case *tengo.Float:
		return v.Value
	case *tengo.String:
		var out float64
		_, _ = fmt.Sscanf(v.Value, "%f", &out)
		return out
	default:
		panic("unsupported type for objectAsFloat")
	}
}
