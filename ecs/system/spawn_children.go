package system

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
)

type SpawnChildrenSystem struct{}

func NewSpawnChildrenSystem() *SpawnChildrenSystem {
	return &SpawnChildrenSystem{}
}

func (s *SpawnChildrenSystem) Update(w *ecs.World) {
	if s == nil || w == nil {
		return
	}

	ecs.ForEach2(w, component.SpawnChildrenComponent.Kind(), component.SpawnChildrenRuntimeComponent.Kind(), func(parent ecs.Entity, cfg *component.SpawnChildren, runtime *component.SpawnChildrenRuntime) {
		if cfg == nil || runtime == nil {
			return
		}
		if runtime.Spawned == nil {
			runtime.Spawned = map[string]uint64{}
		}

		for i, child := range cfg.Children {
			if child.Prefab == "" {
				continue
			}
			key := fmt.Sprintf("%d:%s", i, child.Prefab)

			if existing, ok := runtime.Spawned[key]; ok {
				existingEntity := ecs.Entity(existing)
				if existingEntity.Valid() && ecs.IsAlive(w, existingEntity) {
					if t, ok := ecs.Get(w, existingEntity, component.TransformComponent.Kind()); ok && t != nil {
						t.Parent = uint64(parent)
					}
					continue
				}
			}

			childEntity, err := entity.BuildEntity(w, child.Prefab)
			if err != nil {
				panic("spawn children system: build child entity failed: " + err.Error())
			}

			t, ok := ecs.Get(w, childEntity, component.TransformComponent.Kind())
			if !ok || t == nil {
				t = &component.Transform{ScaleX: 1, ScaleY: 1}
			}
			t.Parent = uint64(parent)
			if err := ecs.Add(w, childEntity, component.TransformComponent.Kind(), t); err != nil {
				panic("spawn children system: set child transform failed: " + err.Error())
			}

			runtime.Spawned[key] = uint64(childEntity)
		}
	})
}
