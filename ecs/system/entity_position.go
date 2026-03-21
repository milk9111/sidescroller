package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func playerWorldPosition(w *ecs.World) (float64, float64, bool) {
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return 0, 0, false
	}
	return entityWorldPosition(w, player)
}

func entityWorldPosition(w *ecs.World, ent ecs.Entity) (float64, float64, bool) {
	if pb, ok := ecs.Get(w, ent, component.PhysicsBodyComponent.Kind()); ok && pb.Body != nil {
		pos := pb.Body.Position()
		return pos.X, pos.Y, true
	}

	if t, ok := ecs.Get(w, ent, component.TransformComponent.Kind()); ok {
		if t.Parent != 0 {
			return t.WorldX, t.WorldY, true
		}
		return t.X, t.Y, true
	}

	return 0, 0, false
}