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

	ecs.ForEach2(w, component.GateComponent.Kind(), component.ArenaNodeComponent.Kind(), func(e ecs.Entity, _ *component.Gate, node *component.ArenaNode) {
		if node == nil {
			return
		}

		rt, ok := ecs.Get(w, e, component.GateRuntimeComponent.Kind())
		if !ok || rt == nil {
			rt = &component.GateRuntime{}
		}

		if !rt.Initialized {
			if sp, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && sp != nil {
				rt.HasSprite = true
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

		if node.Active {
			if rt.HasSprite && !ecs.Has(w, e, component.SpriteComponent.Kind()) {
				template := rt.SpriteTemplate
				_ = ecs.Add(w, e, component.SpriteComponent.Kind(), &template)
			}
			if rt.HasPhysicsBody && !ecs.Has(w, e, component.PhysicsBodyComponent.Kind()) {
				template := rt.PhysicsTemplate
				template.Body = nil
				template.Shape = nil
				_ = ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), &template)
			}
		} else {
			_ = ecs.Remove(w, e, component.SpriteComponent.Kind())
			_ = ecs.Remove(w, e, component.PhysicsBodyComponent.Kind())
		}

		_ = ecs.Add(w, e, component.GateRuntimeComponent.Kind(), rt)
	})
}
