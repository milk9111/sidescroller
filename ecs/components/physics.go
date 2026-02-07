package components

import "github.com/jakecoffman/cp"

// PhysicsBody stores Chipmunk body and shapes for an entity.
type PhysicsBody struct {
	Body        *cp.Body
	Shape       *cp.Shape
	GroundShape *cp.Shape
}
