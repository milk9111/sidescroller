package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type TransformHierarchySystem struct{}

func NewTransformHierarchySystem() *TransformHierarchySystem {
	return &TransformHierarchySystem{}
}

func (s *TransformHierarchySystem) Update(w *ecs.World) {
	if s == nil || w == nil {
		return
	}

	const (
		stateUnvisited = iota
		stateVisiting
		stateResolved
	)

	state := make(map[ecs.Entity]int)

	var resolve func(e ecs.Entity)
	resolve = func(e ecs.Entity) {
		if !e.Valid() || !ecs.IsAlive(w, e) {
			return
		}

		t, ok := ecs.Get(w, e, component.TransformComponent.Kind())
		if !ok || t == nil {
			return
		}

		switch state[e] {
		case stateResolved:
			return
		case stateVisiting:
			setTransformWorldFromLocal(t)
			state[e] = stateResolved
			return
		}

		state[e] = stateVisiting
		if t.Parent != 0 {
			parent := ecs.Entity(t.Parent)
			if parent.Valid() && parent != e && ecs.IsAlive(w, parent) {
				resolve(parent)
				if pt, ok := ecs.Get(w, parent, component.TransformComponent.Kind()); ok && pt != nil {
					applyParentTransform(t, pt)
					state[e] = stateResolved
					return
				}
			}
		}

		setTransformWorldFromLocal(t)
		state[e] = stateResolved
	}

	ecs.ForEach(w, component.TransformComponent.Kind(), func(e ecs.Entity, _ *component.Transform) {
		resolve(e)
	})
}

func setTransformWorldFromLocal(t *component.Transform) {
	if t == nil {
		return
	}
	sx := t.ScaleX
	if sx == 0 {
		sx = 1
	}
	sy := t.ScaleY
	if sy == 0 {
		sy = 1
	}
	t.WorldX = t.X
	t.WorldY = t.Y
	t.WorldScaleX = sx
	t.WorldScaleY = sy
	t.WorldRotation = t.Rotation
}

func applyParentTransform(t *component.Transform, parent *component.Transform) {
	if t == nil || parent == nil {
		return
	}

	localScaleX := t.ScaleX
	if localScaleX == 0 {
		localScaleX = 1
	}
	localScaleY := t.ScaleY
	if localScaleY == 0 {
		localScaleY = 1
	}

	parentScaleX := parent.WorldScaleX
	if parentScaleX == 0 {
		parentScaleX = 1
	}
	parentScaleY := parent.WorldScaleY
	if parentScaleY == 0 {
		parentScaleY = 1
	}

	px := t.X * parentScaleX
	py := t.Y * parentScaleY

	cosR := math.Cos(parent.WorldRotation)
	sinR := math.Sin(parent.WorldRotation)
	rx := px*cosR - py*sinR
	ry := px*sinR + py*cosR

	t.WorldX = parent.WorldX + rx
	t.WorldY = parent.WorldY + ry
	t.WorldRotation = parent.WorldRotation + t.Rotation
	t.WorldScaleX = parentScaleX * localScaleX
	t.WorldScaleY = parentScaleY * localScaleY
}
