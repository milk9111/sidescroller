package component

import (
	"errors"
	"sync/atomic"
)

var (
	ErrEntityNotAlive       = errors.New("ecs: entity not alive")
	ErrNilComponent         = errors.New("ecs: component is nil")
	ErrInvalidComponentKind = errors.New("ecs: invalid component kind")
)

type ComponentKind[T any] struct {
	id ComponentID
}

func NewComponentKind[T any]() ComponentKind[T] {
	return ComponentKind[T]{id: ComponentID(nextComponentID.Add(1))}
}

func (k ComponentKind[T]) ID() ComponentID {
	return k.id
}

func (k ComponentKind[T]) Valid() bool {
	return k.id != 0
}

type ComponentHandle[T any] struct {
	kind ComponentKind[T]
}

func NewComponent[T any]() ComponentHandle[T] {
	return ComponentHandle[T]{kind: NewComponentKind[T]()}
}

func (h ComponentHandle[T]) Kind() ComponentKind[T] {
	return h.kind
}

type ComponentID uint32

var nextComponentID atomic.Uint32
