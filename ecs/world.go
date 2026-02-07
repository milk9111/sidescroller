package ecs

import "github.com/milk9111/sidescroller/ecs/component"

type World struct {
	nextID      entityID
	freeIDs     []entityID
	generations []generation
	alive       map[entityID]struct{}
	components  map[component.ComponentID]map[entityID]any
}

func NewWorld() *World {
	return &World{
		alive:      make(map[entityID]struct{}),
		components: make(map[component.ComponentID]map[entityID]any),
	}
}

func (w *World) CreateEntity() Entity {
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
	w.alive[id] = struct{}{}
	return makeEntity(id, gen)
}

func (w *World) IsAlive(e Entity) bool {
	id := e.id()
	if int(id) >= len(w.generations) {
		return false
	}
	if w.generations[id] != e.generation() {
		return false
	}
	_, ok := w.alive[id]
	return ok
}

func (w *World) DestroyEntity(e Entity) bool {
	if !w.IsAlive(e) {
		return false
	}

	id := e.id()
	for _, store := range w.components {
		delete(store, id)
	}

	delete(w.alive, id)
	w.generations[id]++
	w.freeIDs = append(w.freeIDs, id)
	return true
}

func (w *World) AddComponent(e Entity, kind component.ComponentKind, value any) error {
	if value == nil {
		return component.ErrNilComponent
	}
	if !kind.Valid() {
		return component.ErrInvalidComponentKind
	}
	if !w.IsAlive(e) {
		return component.ErrEntityNotAlive
	}

	store, ok := w.components[kind.ID()]
	if !ok {
		store = make(map[entityID]any)
		w.components[kind.ID()] = store
	}
	store[e.id()] = value
	return nil
}

func (w *World) RemoveComponent(e Entity, kind component.ComponentKind) bool {
	if !w.IsAlive(e) {
		return false
	}
	if !kind.Valid() {
		return false
	}
	store, ok := w.components[kind.ID()]
	if !ok {
		return false
	}
	_, existed := store[e.id()]
	delete(store, e.id())
	return existed
}

func (w *World) GetComponent(e Entity, kind component.ComponentKind) (any, bool) {
	if !w.IsAlive(e) {
		return nil, false
	}
	if !kind.Valid() {
		return nil, false
	}
	store, ok := w.components[kind.ID()]
	if !ok {
		return nil, false
	}
	component, ok := store[e.id()]
	return component, ok
}

func (w *World) HasComponent(e Entity, kind component.ComponentKind) bool {
	if !w.IsAlive(e) {
		return false
	}
	if !kind.Valid() {
		return false
	}
	store, ok := w.components[kind.ID()]
	if !ok {
		return false
	}
	_, ok = store[e.id()]
	return ok
}

func (w *World) Entities() []Entity {
	entities := make([]Entity, 0, len(w.alive))
	for id := range w.alive {
		entities = append(entities, makeEntity(id, w.generations[id]))
	}
	return entities
}

func (w *World) Query(kinds ...component.ComponentKind) []Entity {
	if len(kinds) == 0 {
		return w.Entities()
	}

	var baseKind component.ComponentKind
	var baseStore map[entityID]any
	for _, kind := range kinds {
		if !kind.Valid() {
			return nil
		}
		store, ok := w.components[kind.ID()]
		if !ok || len(store) == 0 {
			return nil
		}
		if baseStore == nil || len(store) < len(baseStore) {
			baseStore = store
			baseKind = kind
		}
	}

	result := make([]Entity, 0, len(baseStore))
	for id := range baseStore {
		if _, alive := w.alive[id]; !alive {
			continue
		}
		match := true
		for _, kind := range kinds {
			if kind == baseKind {
				continue
			}
			store := w.components[kind.ID()]
			if _, ok := store[id]; !ok {
				match = false
				break
			}
		}
		if match {
			result = append(result, makeEntity(id, w.generations[id]))
		}
	}

	return result
}
