package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type PickupHoverSystem struct{}

func NewPickupHoverSystem() *PickupHoverSystem { return &PickupHoverSystem{} }

func (s *PickupHoverSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ecs.ForEach2(w, component.PickupComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, pickup *component.Pickup, t *component.Transform) {
		if pickup == nil || t == nil {
			return
		}

		if !pickup.Initialized {
			pickup.BaseY = t.Y
			pickup.Initialized = true
			if pickup.BobAmplitude == 0 {
				pickup.BobAmplitude = 4
			}
			if pickup.BobSpeed == 0 {
				pickup.BobSpeed = 0.08
			}
		}

		pickup.BobPhase += pickup.BobSpeed
		t.Y = pickup.BaseY + math.Sin(pickup.BobPhase)*pickup.BobAmplitude
		_ = ecs.Add(w, e, component.TransformComponent.Kind(), t)
		_ = ecs.Add(w, e, component.PickupComponent.Kind(), pickup)
	})
}
