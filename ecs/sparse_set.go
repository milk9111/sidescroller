package ecs

// SparseSet is a cache-friendly storage for components keyed by Entity ID.
// It stores components as `any` to avoid generics in older toolchains.
type SparseSet struct {
	denseEntities []int
	denseValues   []any
	sparse        []int
}

// Has returns true if the entity id exists in the set.
func (s *SparseSet) Has(id int) bool {
	if s == nil || id <= 0 || id-1 >= len(s.sparse) {
		return false
	}
	idx := s.sparse[id-1]
	return idx >= 0 && idx < len(s.denseEntities) && s.denseEntities[idx] == id
}

// Get returns the component for id, or nil.
func (s *SparseSet) Get(id int) any {
	if !s.Has(id) {
		return nil
	}
	idx := s.sparse[id-1]
	return s.denseValues[idx]
}

// Set inserts or updates a component for id.
func (s *SparseSet) Set(id int, v any) {
	if s == nil || id <= 0 {
		return
	}
	if id-1 >= len(s.sparse) {
		grow := id - len(s.sparse)
		for i := 0; i < grow; i++ {
			s.sparse = append(s.sparse, -1)
		}
	}
	if s.Has(id) {
		idx := s.sparse[id-1]
		s.denseValues[idx] = v
		return
	}
	s.denseEntities = append(s.denseEntities, id)
	s.denseValues = append(s.denseValues, v)
	s.sparse[id-1] = len(s.denseEntities) - 1
}

// Remove deletes the component for id if present.
func (s *SparseSet) Remove(id int) {
	if s == nil || !s.Has(id) {
		return
	}
	idx := s.sparse[id-1]
	last := len(s.denseEntities) - 1
	lastID := s.denseEntities[last]

	s.denseEntities[idx] = s.denseEntities[last]
	s.denseValues[idx] = s.denseValues[last]
	s.sparse[lastID-1] = idx

	s.denseEntities = s.denseEntities[:last]
	s.denseValues = s.denseValues[:last]
	s.sparse[id-1] = -1
}

// Entities returns the dense entity id list.
func (s *SparseSet) Entities() []int {
	if s == nil {
		return nil
	}
	return s.denseEntities
}

// Values returns the dense component list.
func (s *SparseSet) Values() []any {
	if s == nil {
		return nil
	}
	return s.denseValues
}
