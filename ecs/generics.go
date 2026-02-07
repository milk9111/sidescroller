package ecs

import "github.com/milk9111/sidescroller/ecs/component"

func Add[T any](w *World, e Entity, handle component.ComponentHandle[T], value T) error {
	return w.AddComponent(e, handle.Kind(), value)
}

func Remove[T any](w *World, e Entity, handle component.ComponentHandle[T]) bool {
	return w.RemoveComponent(e, handle.Kind())
}

func Has[T any](w *World, e Entity, handle component.ComponentHandle[T]) bool {
	return w.HasComponent(e, handle.Kind())
}

func Get[T any](w *World, e Entity, handle component.ComponentHandle[T]) (T, bool) {
	var zero T
	value, ok := w.GetComponent(e, handle.Kind())
	if !ok {
		return zero, false
	}
	cast, ok := value.(T)
	if !ok {
		return zero, false
	}
	return cast, true
}
