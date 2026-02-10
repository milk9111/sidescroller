package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AnchorSystem struct{}

func NewAnchorSystem() *AnchorSystem { return &AnchorSystem{} }

func (s *AnchorSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	entities := w.Query(component.AnchorComponent.Kind(), component.TransformComponent.Kind())
	for _, e := range entities {
		a, ok := ecs.Get(w, e, component.AnchorComponent)
		if !ok {
			continue
		}
		t, ok := ecs.Get(w, e, component.TransformComponent)
		if !ok {
			continue
		}

		dx := a.TargetX - t.X
		dy := a.TargetY - t.Y
		dist := math.Hypot(dx, dy)
		if dist <= 0 {
			// already there: remove Anchor component
			w.RemoveComponent(e, component.AnchorComponent.Kind())
			continue
		}

		step := a.Speed
		if step <= 0 {
			step = 10
		}

		if dist <= step {
			t.X = a.TargetX
			t.Y = a.TargetY
			if err := ecs.Add(w, e, component.TransformComponent, t); err != nil {
				panic("anchor system: update transform: " + err.Error())
			}
			// reached target; remove Anchor component to stop updating
			w.RemoveComponent(e, component.AnchorComponent.Kind())
			continue
		}

		nx := dx / dist
		ny := dy / dist
		t.X += nx * step
		t.Y += ny * step
		if err := ecs.Add(w, e, component.TransformComponent, t); err != nil {
			panic("anchor system: update transform: " + err.Error())
		}
	}
}
