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

type ComponentID uint32

type ComponentKind struct {
	id ComponentID
}

type ComponentHandle[T any] struct {
	kind ComponentKind
}

var nextComponentID atomic.Uint32

func NewComponentKind() ComponentKind {
	return ComponentKind{id: ComponentID(nextComponentID.Add(1))}
}

func NewComponent[T any]() ComponentHandle[T] {
	return ComponentHandle[T]{kind: NewComponentKind()}
}

func (k ComponentKind) ID() ComponentID {
	return k.id
}

func (k ComponentKind) Valid() bool {
	return k.id != 0
}

func (h ComponentHandle[T]) Kind() ComponentKind {
	return h.kind
}
