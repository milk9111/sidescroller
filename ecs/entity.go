package ecs

import "strconv"

type Entity uint64

type entityID uint32
type generation uint32

const entityIDBits = 32

func makeEntity(id entityID, gen generation) Entity {
	return Entity(uint64(gen)<<entityIDBits | uint64(id))
}

func (e Entity) id() entityID {
	return entityID(uint32(e))
}

func (e Entity) generation() generation {
	return generation(uint32(uint64(e) >> entityIDBits))
}

func (e Entity) String() string {
	return strconv.FormatUint(uint64(e), 10)
}

func (e Entity) Valid() bool {
	return e > 0
}
