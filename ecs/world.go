package ecs

import (
	"errors"

	"github.com/milk9111/sidescroller/ecs/component"
)

var errSparseComponentTypeMismatch = errors.New("ecs: component type mismatch")

type sparseSet[T any] struct {
	sparse []int
	dense  []entityID
	data   []*T
}

func (s *sparseSet[T]) len() int {
	return len(s.dense)
}

func (s *sparseSet[T]) ids() []entityID {
	return s.dense
}

func (s *sparseSet[T]) has(id entityID) bool {
	idx := int(id)
	if idx < 0 || idx >= len(s.sparse) {
		return false
	}
	denseIndex := s.sparse[idx]
	return denseIndex >= 0 && denseIndex < len(s.dense) && s.dense[denseIndex] == id
}

func (s *sparseSet[T]) get(id entityID) (*T, bool) {
	idx := int(id)
	if idx < 0 || idx >= len(s.sparse) {
		return nil, false
	}
	denseIndex := s.sparse[idx]
	if denseIndex < 0 || denseIndex >= len(s.dense) || s.dense[denseIndex] != id {
		return nil, false
	}
	return s.data[denseIndex], true
}

func (s *sparseSet[T]) set(id entityID, value *T) {
	s.ensureSparse(id)
	idx := s.sparse[int(id)]
	if idx >= 0 {
		s.data[idx] = value
		return
	}

	s.sparse[int(id)] = len(s.dense)
	s.dense = append(s.dense, id)
	s.data = append(s.data, value)
}

func (s *sparseSet[T]) remove(id entityID) bool {
	idx := int(id)
	if idx < 0 || idx >= len(s.sparse) {
		return false
	}
	denseIndex := s.sparse[idx]
	if denseIndex < 0 || denseIndex >= len(s.dense) || s.dense[denseIndex] != id {
		return false
	}

	last := len(s.dense) - 1
	lastID := s.dense[last]

	s.dense[denseIndex] = lastID
	s.data[denseIndex] = s.data[last]
	s.sparse[int(lastID)] = denseIndex

	s.dense = s.dense[:last]

	s.data[last] = nil
	s.data = s.data[:last]
	s.sparse[idx] = -1

	return true
}

func (s *sparseSet[T]) ensureSparse(id entityID) {
	required := int(id) + 1
	if required <= len(s.sparse) {
		return
	}

	oldLen := len(s.sparse)
	s.sparse = append(s.sparse, make([]int, required-oldLen)...)
	for i := oldLen; i < len(s.sparse); i++ {
		s.sparse[i] = -1
	}
}

type sparseComponentStore[T any] struct {
	items sparseSet[T]
}

func (s *sparseComponentStore[T]) len() int {
	return s.items.len()
}

func (s *sparseComponentStore[T]) ids() []entityID {
	return s.items.ids()
}

func (s *sparseComponentStore[T]) remove(id entityID) bool {
	return s.items.remove(id)
}

func (s *sparseComponentStore[T]) has(id entityID) bool {
	return s.items.has(id)
}

func (s *sparseComponentStore[T]) get(id entityID) (*T, bool) {
	return s.items.get(id)
}

func (s *sparseComponentStore[T]) set(id entityID, value *T) {
	s.items.set(id, value)
}

type componentStore interface {
	len() int
	ids() []entityID
	remove(entityID) bool
	has(id entityID) bool
}

type World struct {
	nextID      entityID
	freeIDs     []entityID
	generations []generation
	alive       sparseSet[struct{}]
	components  map[component.ComponentID]any
}

func NewWorld() *World {
	return &World{
		components: make(map[component.ComponentID]any),
	}
}

func CreateEntity(w *World) Entity {
	var id entityID
	if n := len(w.freeIDs); n > 0 {
		id = w.freeIDs[n-1]
		w.freeIDs = w.freeIDs[:n-1]
	} else {
		id = w.nextID
		w.nextID++
		if int(id) >= len(w.generations) {
			w.generations = append(w.generations, 0)
		}
	}

	gen := w.generations[id]
	w.alive.set(id, &struct{}{})
	return makeEntity(id, gen)
}

func IsAlive(w *World, e Entity) bool {
	id := e.id()
	if int(id) >= len(w.generations) {
		return false
	}
	if w.generations[id] != e.generation() {
		return false
	}
	return w.alive.has(id)
}

func DestroyEntity(w *World, e Entity) bool {
	if !IsAlive(w, e) {
		return false
	}

	id := e.id()
	for _, v := range w.components {
		store, ok := v.(componentStore)
		if !ok {
			panic("ecs: component store type mismatch")
		}

		store.remove(id)
	}

	w.alive.remove(id)
	w.generations[id]++
	w.freeIDs = append(w.freeIDs, id)

	return true
}

func Add[T any](w *World, e Entity, kind component.ComponentKind[T], value *T) error {
	if !kind.Valid() {
		return component.ErrInvalidComponentKind
	}

	if !IsAlive(w, e) {
		return component.ErrEntityNotAlive
	}

	v, ok := w.components[kind.ID()]
	var store *sparseComponentStore[T]
	if !ok {
		store = &sparseComponentStore[T]{}
		w.components[kind.ID()] = store
	} else {
		store, ok = v.(*sparseComponentStore[T])
		if !ok {
			return errSparseComponentTypeMismatch
		}
	}

	store.set(e.id(), value)
	return nil
}

func Remove[T any](w *World, e Entity, kind component.ComponentKind[T]) bool {
	if !IsAlive(w, e) || !kind.Valid() {
		return false
	}

	v, ok := w.components[kind.ID()]
	if !ok {
		return false
	}

	store, ok := v.(*sparseComponentStore[T])
	if !ok {
		return false
	}

	return store.remove(e.id())
}

func Get[T any](w *World, e Entity, kind component.ComponentKind[T]) (*T, bool) {
	if !IsAlive(w, e) || !kind.Valid() {
		return nil, false
	}

	v, ok := w.components[kind.ID()]
	if !ok {
		return nil, false
	}

	store, ok := v.(*sparseComponentStore[T])
	if !ok {
		return nil, false
	}

	return store.get(e.id())
}

func Has[T any](w *World, e Entity, kind component.ComponentKind[T]) bool {
	if !IsAlive(w, e) || !kind.Valid() {
		return false
	}

	v, ok := w.components[kind.ID()]
	if !ok {
		return false
	}

	store, ok := v.(*sparseComponentStore[T])
	if !ok {
		return false
	}

	return store.has(e.id())
}

func Entities(w *World) []Entity {
	ids := w.alive.ids()
	entities := make([]Entity, 0, len(ids))
	for _, id := range ids {
		entities = append(entities, makeEntity(id, w.generations[id]))
	}
	return entities
}

func ForEach[A any](w *World, k component.ComponentKind[A], fn func(Entity, *A)) {
	if !k.Valid() {
		return
	}
	v, ok := w.components[k.ID()]
	if !ok {
		return
	}
	s, ok := v.(*sparseComponentStore[A])
	if !ok || s.len() == 0 {
		return
	}

	for _, id := range s.ids() {
		if !w.alive.has(id) {
			continue
		}
		a, _ := s.get(id)
		fn(makeEntity(id, w.generations[id]), a)
	}
}

func ForEach2[A, B any](w *World, k1 component.ComponentKind[A], k2 component.ComponentKind[B], fn func(Entity, *A, *B)) {
	if !k1.Valid() || !k2.Valid() {
		return
	}

	v1, ok := w.components[k1.ID()]
	if !ok {
		return
	}
	s1, ok := v1.(*sparseComponentStore[A])
	if !ok || s1.len() == 0 {
		return
	}

	v2, ok := w.components[k2.ID()]
	if !ok {
		return
	}
	s2, ok := v2.(*sparseComponentStore[B])
	if !ok || s2.len() == 0 {
		return
	}

	// iterate the smaller store
	if s2.len() < s1.len() {
		for _, id := range s2.ids() {
			if !w.alive.has(id) {
				continue
			}
			if !s1.has(id) {
				continue
			}
			a, _ := s1.get(id)
			b, _ := s2.get(id)
			fn(makeEntity(id, w.generations[id]), a, b)
		}
		return
	}

	for _, id := range s1.ids() {
		if !w.alive.has(id) {
			continue
		}
		if !s2.has(id) {
			continue
		}
		a, _ := s1.get(id)
		b, _ := s2.get(id)
		fn(makeEntity(id, w.generations[id]), a, b)
	}
}

func ForEach3[A, B, C any](w *World, k1 component.ComponentKind[A], k2 component.ComponentKind[B], k3 component.ComponentKind[C], fn func(Entity, *A, *B, *C)) {
	if !k1.Valid() || !k2.Valid() || !k3.Valid() {
		return
	}

	v1, ok := w.components[k1.ID()]
	if !ok {
		return
	}
	s1, ok := v1.(*sparseComponentStore[A])
	if !ok || s1.len() == 0 {
		return
	}

	v2, ok := w.components[k2.ID()]
	if !ok {
		return
	}
	s2, ok := v2.(*sparseComponentStore[B])
	if !ok || s2.len() == 0 {
		return
	}

	v3, ok := w.components[k3.ID()]
	if !ok {
		return
	}
	s3, ok := v3.(*sparseComponentStore[C])
	if !ok || s3.len() == 0 {
		return
	}

	stores := []componentStore{s1, s2, s3}
	min := 0
	for i := 1; i < len(stores); i++ {
		if stores[i].len() < stores[min].len() {
			min = i
		}
	}

	var base componentStore = stores[min]
	others := make([]componentStore, 0, 2)
	for i := 0; i < len(stores); i++ {
		if i == min {
			continue
		}
		others = append(others, stores[i])
	}

	for _, id := range base.ids() {
		if !w.alive.has(id) {
			continue
		}
		ok := true
		for _, o := range others {
			if !o.has(id) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		a, _ := s1.get(id)
		b, _ := s2.get(id)
		c, _ := s3.get(id)
		fn(makeEntity(id, w.generations[id]), a, b, c)
	}
}

func ForEach4[A, B, C, D any](w *World, k1 component.ComponentKind[A], k2 component.ComponentKind[B], k3 component.ComponentKind[C], k4 component.ComponentKind[D], fn func(Entity, *A, *B, *C, *D)) {
	if !k1.Valid() || !k2.Valid() || !k3.Valid() || !k4.Valid() {
		return
	}

	v1, ok := w.components[k1.ID()]
	if !ok {
		return
	}
	s1, ok := v1.(*sparseComponentStore[A])
	if !ok || s1.len() == 0 {
		return
	}

	v2, ok := w.components[k2.ID()]
	if !ok {
		return
	}
	s2, ok := v2.(*sparseComponentStore[B])
	if !ok || s2.len() == 0 {
		return
	}

	v3, ok := w.components[k3.ID()]
	if !ok {
		return
	}
	s3, ok := v3.(*sparseComponentStore[C])
	if !ok || s3.len() == 0 {
		return
	}

	v4, ok := w.components[k4.ID()]
	if !ok {
		return
	}
	s4, ok := v4.(*sparseComponentStore[D])
	if !ok || s4.len() == 0 {
		return
	}

	stores := []componentStore{s1, s2, s3, s4}
	min := 0
	for i := 1; i < len(stores); i++ {
		if stores[i].len() < stores[min].len() {
			min = i
		}
	}

	base := stores[min]
	others := make([]componentStore, 0, 3)
	for i := 0; i < len(stores); i++ {
		if i == min {
			continue
		}
		others = append(others, stores[i])
	}

	for _, id := range base.ids() {
		if !w.alive.has(id) {
			continue
		}
		ok := true
		for _, o := range others {
			if !o.has(id) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		a, _ := s1.get(id)
		b, _ := s2.get(id)
		c, _ := s3.get(id)
		d, _ := s4.get(id)
		fn(makeEntity(id, w.generations[id]), a, b, c, d)
	}
}

// ForEach7 iterates entities that have all seven component kinds and calls fn
func ForEach7[A, B, C, D, E, F, G any](w *World, k1 component.ComponentKind[A], k2 component.ComponentKind[B], k3 component.ComponentKind[C], k4 component.ComponentKind[D], k5 component.ComponentKind[E], k6 component.ComponentKind[F], k7 component.ComponentKind[G], fn func(Entity, *A, *B, *C, *D, *E, *F, *G)) {
	if !k1.Valid() || !k2.Valid() || !k3.Valid() || !k4.Valid() || !k5.Valid() || !k6.Valid() || !k7.Valid() {
		return
	}

	v1, ok := w.components[k1.ID()]
	if !ok {
		return
	}
	s1, ok := v1.(*sparseComponentStore[A])
	if !ok || s1.len() == 0 {
		return
	}

	v2, ok := w.components[k2.ID()]
	if !ok {
		return
	}
	s2, ok := v2.(*sparseComponentStore[B])
	if !ok || s2.len() == 0 {
		return
	}

	v3, ok := w.components[k3.ID()]
	if !ok {
		return
	}
	s3, ok := v3.(*sparseComponentStore[C])
	if !ok || s3.len() == 0 {
		return
	}

	v4, ok := w.components[k4.ID()]
	if !ok {
		return
	}
	s4, ok := v4.(*sparseComponentStore[D])
	if !ok || s4.len() == 0 {
		return
	}

	v5, ok := w.components[k5.ID()]
	if !ok {
		return
	}
	s5, ok := v5.(*sparseComponentStore[E])
	if !ok || s5.len() == 0 {
		return
	}

	v6, ok := w.components[k6.ID()]
	if !ok {
		return
	}
	s6, ok := v6.(*sparseComponentStore[F])
	if !ok || s6.len() == 0 {
		return
	}

	v7, ok := w.components[k7.ID()]
	if !ok {
		return
	}
	s7, ok := v7.(*sparseComponentStore[G])
	if !ok || s7.len() == 0 {
		return
	}

	stores := []componentStore{s1, s2, s3, s4, s5, s6, s7}
	min := 0
	for i := 1; i < len(stores); i++ {
		if stores[i].len() < stores[min].len() {
			min = i
		}
	}

	base := stores[min]
	others := make([]componentStore, 0, 6)
	for i := 0; i < len(stores); i++ {
		if i == min {
			continue
		}
		others = append(others, stores[i])
	}

	for _, id := range base.ids() {
		if !w.alive.has(id) {
			continue
		}
		ok := true
		for _, o := range others {
			if !o.has(id) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		a, _ := s1.get(id)
		b, _ := s2.get(id)
		c, _ := s3.get(id)
		d, _ := s4.get(id)
		e, _ := s5.get(id)
		f, _ := s6.get(id)
		g, _ := s7.get(id)
		fn(makeEntity(id, w.generations[id]), a, b, c, d, e, f, g)
	}
}

// ForEach8 iterates entities that have all eight component kinds and calls fn
func ForEach8[A, B, C, D, E, F, G, H any](w *World, k1 component.ComponentKind[A], k2 component.ComponentKind[B], k3 component.ComponentKind[C], k4 component.ComponentKind[D], k5 component.ComponentKind[E], k6 component.ComponentKind[F], k7 component.ComponentKind[G], k8 component.ComponentKind[H], fn func(Entity, *A, *B, *C, *D, *E, *F, *G, *H)) {
	if !k1.Valid() || !k2.Valid() || !k3.Valid() || !k4.Valid() || !k5.Valid() || !k6.Valid() || !k7.Valid() || !k8.Valid() {
		return
	}

	v1, ok := w.components[k1.ID()]
	if !ok {
		return
	}
	s1, ok := v1.(*sparseComponentStore[A])
	if !ok || s1.len() == 0 {
		return
	}

	v2, ok := w.components[k2.ID()]
	if !ok {
		return
	}
	s2, ok := v2.(*sparseComponentStore[B])
	if !ok || s2.len() == 0 {
		return
	}

	v3, ok := w.components[k3.ID()]
	if !ok {
		return
	}
	s3, ok := v3.(*sparseComponentStore[C])
	if !ok || s3.len() == 0 {
		return
	}

	v4, ok := w.components[k4.ID()]
	if !ok {
		return
	}
	s4, ok := v4.(*sparseComponentStore[D])
	if !ok || s4.len() == 0 {
		return
	}

	v5, ok := w.components[k5.ID()]
	if !ok {
		return
	}
	s5, ok := v5.(*sparseComponentStore[E])
	if !ok || s5.len() == 0 {
		return
	}

	v6, ok := w.components[k6.ID()]
	if !ok {
		return
	}
	s6, ok := v6.(*sparseComponentStore[F])
	if !ok || s6.len() == 0 {
		return
	}

	v7, ok := w.components[k7.ID()]
	if !ok {
		return
	}
	s7, ok := v7.(*sparseComponentStore[G])
	if !ok || s7.len() == 0 {
		return
	}

	v8, ok := w.components[k8.ID()]
	if !ok {
		return
	}
	s8, ok := v8.(*sparseComponentStore[H])
	if !ok || s8.len() == 0 {
		return
	}

	stores := []componentStore{s1, s2, s3, s4, s5, s6, s7, s8}
	min := 0
	for i := 1; i < len(stores); i++ {
		if stores[i].len() < stores[min].len() {
			min = i
		}
	}

	base := stores[min]
	others := make([]componentStore, 0, 7)
	for i := 0; i < len(stores); i++ {
		if i == min {
			continue
		}
		others = append(others, stores[i])
	}

	for _, id := range base.ids() {
		if !w.alive.has(id) {
			continue
		}
		ok := true
		for _, o := range others {
			if !o.has(id) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		a, _ := s1.get(id)
		b, _ := s2.get(id)
		c, _ := s3.get(id)
		d, _ := s4.get(id)
		e, _ := s5.get(id)
		f, _ := s6.get(id)
		g, _ := s7.get(id)
		h, _ := s8.get(id)
		fn(makeEntity(id, w.generations[id]), a, b, c, d, e, f, g, h)
	}
}

// Query2 returns the entities that have both component kinds.
func Query2[A, B any](w *World, k1 component.ComponentKind[A], k2 component.ComponentKind[B]) []Entity {
	if !k1.Valid() || !k2.Valid() {
		return nil
	}
	v1, ok := w.components[k1.ID()]
	if !ok {
		return nil
	}
	s1, ok := v1.(*sparseComponentStore[A])
	if !ok || s1.len() == 0 {
		return nil
	}
	v2, ok := w.components[k2.ID()]
	if !ok {
		return nil
	}
	s2, ok := v2.(*sparseComponentStore[B])
	if !ok || s2.len() == 0 {
		return nil
	}

	var base componentStore = s1
	var other componentStore = s2
	if s2.len() < base.len() {
		base = s2
		other = s1
	}

	result := make([]Entity, 0, base.len())
	for _, id := range base.ids() {
		if !w.alive.has(id) {
			continue
		}
		if !other.has(id) {
			continue
		}
		result = append(result, makeEntity(id, w.generations[id]))
	}

	return result
}

// First returns the first alive entity that has component kind k, or false if none.
func First[A any](w *World, k component.ComponentKind[A]) (Entity, bool) {
	var zero Entity
	if !k.Valid() {
		return zero, false
	}
	v, ok := w.components[k.ID()]
	if !ok {
		return zero, false
	}
	s, ok := v.(*sparseComponentStore[A])
	if !ok || s.len() == 0 {
		return zero, false
	}

	for _, id := range s.ids() {
		if !w.alive.has(id) {
			continue
		}
		return makeEntity(id, w.generations[id]), true
	}
	return zero, false
}
