package component

// AnchorDetachRequest is a marker component indicating that the anchor
// should be detached from any physics constraints and disabled by the
// PhysicsSystem.
type AnchorDetachRequest struct{}

var AnchorDetachRequestComponent = NewComponent[AnchorDetachRequest]()
