package ecs

// entityStore tracks entity generations and free ids.
type entityStore struct {
	nextID int
	gen    []int
	free   []int
}

func (s *entityStore) create() Entity {
	if s == nil {
		return Entity{}
	}
	var id int
	if len(s.free) > 0 {
		id = s.free[len(s.free)-1]
		s.free = s.free[:len(s.free)-1]
	} else {
		s.nextID++
		id = s.nextID
		if id >= len(s.gen) {
			s.gen = append(s.gen, 0)
		}
	}
	if id >= len(s.gen) {
		s.gen = append(s.gen, 0)
	}
	return Entity{ID: id, Gen: s.gen[id-1]}
}

func (s *entityStore) destroy(e Entity) {
	if s == nil || e.ID <= 0 || e.ID > len(s.gen) {
		return
	}
	idx := e.ID - 1
	if s.gen[idx] != e.Gen {
		return
	}
	s.gen[idx]++
	s.free = append(s.free, e.ID)
}

func (s *entityStore) isAlive(e Entity) bool {
	if s == nil || e.ID <= 0 || e.ID > len(s.gen) {
		return false
	}
	return s.gen[e.ID-1] == e.Gen
}
