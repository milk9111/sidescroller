package component

import "github.com/jakecoffman/cp"

// PhysicsBody stores Chipmunk2D runtime data and collider configuration.
type PhysicsBody struct {
	Body         *cp.Body
	Shape        *cp.Shape
	Width        float64
	Height       float64
	Radius       float64
	Mass         float64
	Friction     float64
	Elasticity   float64
	Static       bool
	AlignTopLeft bool
}

var PhysicsBodyComponent = NewComponent[PhysicsBody]()
