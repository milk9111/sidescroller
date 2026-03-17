package component

import "github.com/jakecoffman/cp"

// PhysicsBody stores Chipmunk2D runtime data and collider configuration.
type PhysicsBody struct {
	Body          *cp.Body
	Shape         *cp.Shape
	Disabled      bool
	SegmentStartX float64
	SegmentStartY float64
	SegmentEndX   float64
	SegmentEndY   float64
	SegmentRadius float64
	Width         float64
	Height        float64
	Radius        float64
	Mass          float64
	Friction      float64
	Elasticity    float64
	Static        bool
	AlignTopLeft  bool
	OffsetX       float64
	OffsetY       float64
}

var PhysicsBodyComponent = NewComponent[PhysicsBody]()
