package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type GateSystem struct{}

func NewGateSystem() *GateSystem { return &GateSystem{} }

func (s *GateSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	toDestroy := make([]ecs.Entity, 0)

	ecs.ForEach(w, component.GateComponent.Kind(), func(e ecs.Entity, _ *component.Gate) {
		rt, ok := ecs.Get(w, e, component.GateRuntimeComponent.Kind())
		if !ok || rt == nil {
			rt = &component.GateRuntime{}
		}

		if !rt.Initialized {
			if sp, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && sp != nil {
				rt.HasSprite = true
				rt.SpriteWasDisabled = sp.Disabled
				rt.SpriteTemplate = *sp
			}
			if body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); ok && body != nil {
				template := *body
				template.Body = nil
				template.Shape = nil
				rt.HasPhysicsBody = true
				rt.PhysicsTemplate = template
			}
			rt.Initialized = true
		}

		spriteDisabled := true
		if sp, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && sp != nil {
			spriteDisabled = sp.Disabled
		}
		if rt.HasSprite && spriteDisabled && !rt.SpriteWasDisabled {
			recordLevelEntityState(w, e, component.PersistedLevelEntityStateUsed)
			toDestroy = append(toDestroy, e)
			return
		}

		node, _ := ecs.Get(w, e, component.ArenaNodeComponent.Kind())

		if node != nil && node.Active {
			if rt.HasSprite && !ecs.Has(w, e, component.SpriteComponent.Kind()) {
				template := rt.SpriteTemplate
				template.Disabled = false
				_ = ecs.Add(w, e, component.SpriteComponent.Kind(), &template)
			}
			if rt.HasPhysicsBody && !ecs.Has(w, e, component.PhysicsBodyComponent.Kind()) {
				template := rt.PhysicsTemplate
				template.Body = nil
				template.Shape = nil
				_ = ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), &template)
			}
		}

		if sp, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && sp != nil {
			rt.SpriteWasDisabled = sp.Disabled
		} else {
			rt.SpriteWasDisabled = true
		}

		_ = ecs.Add(w, e, component.GateRuntimeComponent.Kind(), rt)
	})

	for _, e := range toDestroy {
		if ecs.IsAlive(w, e) {
			ecs.DestroyEntity(w, e)
		}
	}
}
