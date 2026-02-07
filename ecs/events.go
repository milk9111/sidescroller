package ecs

// Event is a generic ECS event payload.
type Event struct {
	Type string
	Data any
}

// CollisionEventKind identifies collision event types.
type CollisionEventKind string

const (
	CollisionEventGrounded  CollisionEventKind = "grounded"
	CollisionEventHitHazard CollisionEventKind = "hazard"
)

// CollisionEvent is emitted when collision state changes.
type CollisionEvent struct {
	Entity Entity
	Kind   CollisionEventKind
}

// EventQueue is a simple FIFO queue.
type EventQueue struct {
	items []Event
}

// Push adds an event.
func (q *EventQueue) Push(evt Event) {
	if q == nil {
		return
	}
	q.items = append(q.items, evt)
}

// Drain returns all events and clears the queue.
func (q *EventQueue) Drain() []Event {
	if q == nil || len(q.items) == 0 {
		return nil
	}
	out := q.items
	q.items = nil
	return out
}

func (q *EventQueue) flush() {
	if q == nil {
		return
	}
	q.items = nil
}
