package component

import "github.com/jakecoffman/cp"

// AnchorJoint stores constraint handles for the anchor entity.
type AnchorJoint struct {
	Slide *cp.Constraint
	Pivot *cp.Constraint
	Pin   *cp.Constraint
}

var AnchorJointComponent = NewComponent[AnchorJoint]()
