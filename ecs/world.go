package ecs

// System updates a world each frame.
type System interface {
	Update(w *World)
}

// World owns entities, components, and system order.
type World struct {
	entities entityStore
	systems  []System
	events   EventQueue

	transforms   *SparseSet
	sprites      *SparseSet
	animators    *SparseSet
	velocities   *SparseSet
	accels       *SparseSet
	gravities    *SparseSet
	colliders    *SparseSet
	grounders    *SparseSet
	physBodies   *SparseSet
	collStates   *SparseSet
	healths      *SparseSet
	dealers      *SparseSet
	hurtboxes    *SparseSet
	aiStates     *SparseSet
	pathing      *SparseSet
	inputs       *SparseSet
	cameras      *SparseSet
	cameraStates *SparseSet
	playerCtrls  *SparseSet
	pickups      *SparseSet
	bullets      *SparseSet

	physicsWorld *PhysicsWorld
}

// NewWorld creates an empty ECS world.
func NewWorld() *World {
	return &World{}
}

// CreateEntity allocates a new entity.
func (w *World) CreateEntity() Entity {
	return w.entities.create()
}

// DestroyEntity marks an entity as dead.
func (w *World) DestroyEntity(e Entity) {
	w.entities.destroy(e)
}

// IsAlive reports whether an entity handle is valid.
func (w *World) IsAlive(e Entity) bool {
	return w.entities.isAlive(e)
}

// AddSystem appends a system to the update order.
func (w *World) AddSystem(s System) {
	if s == nil {
		return
	}
	w.systems = append(w.systems, s)
}

// Update runs all systems once.
func (w *World) Update() {
	if w == nil {
		return
	}
	for _, s := range w.systems {
		if s != nil {
			s.Update(w)
		}
	}
	w.events.flush()
}

// Events returns the world event queue.
func (w *World) Events() *EventQueue {
	if w == nil {
		return nil
	}
	return &w.events
}

// SetPhysicsWorld attaches a physics world to this ECS world.
func (w *World) SetPhysicsWorld(pw *PhysicsWorld) {
	if w == nil {
		return
	}
	w.physicsWorld = pw
}

// PhysicsWorld returns the attached physics world, if any.
func (w *World) PhysicsWorld() *PhysicsWorld {
	if w == nil {
		return nil
	}
	return w.physicsWorld
}
