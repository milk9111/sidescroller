package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type TriggerSystem struct{}

func NewTriggerSystem() *TriggerSystem { return &TriggerSystem{} }

func (ts *TriggerSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	playerBounds, ok := playerAABB(w, player)
	if !ok {
		return
	}

	ecs.ForEach2(w, component.TriggerComponent.Kind(), component.TransformComponent.Kind(), func(ent ecs.Entity, trigger *component.Trigger, transform *component.Transform) {
		if trigger == nil || transform == nil || trigger.Disabled {
			return
		}

		if !aabbIntersects(playerBounds, triggerAABB(transform, trigger)) {
			return
		}

		if EmitEntitySignal(w, ent, player, "on_trigger_entered") {
			trigger.Disabled = true
		}
	})
}

func triggerAABB(transform *component.Transform, trigger *component.Trigger) aabb {
	if transform == nil || trigger == nil {
		return aabb{}
	}
	width := trigger.Bounds.W
	height := trigger.Bounds.H
	if width <= 0 {
		width = tileSize
	}
	if height <= 0 {
		height = tileSize
	}
	return aabb{
		x: transform.X + trigger.Bounds.X,
		y: transform.Y + trigger.Bounds.Y,
		w: width,
		h: height,
	}
}
