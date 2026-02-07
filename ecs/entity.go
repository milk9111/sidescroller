package ecs

// Entity is a handle to an entity id with generation.
type Entity struct {
	ID  int
	Gen int
}

func (e Entity) IsZero() bool {
	return e.ID == 0
}
