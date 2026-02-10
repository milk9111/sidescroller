package component

// AnchorPendingDestroy is a tag to indicate an anchor entity should be
// cleaned up by the PhysicsSystem (which owns the cp.Space) before being
// removed from the world.
type AnchorPendingDestroy struct{}

var AnchorPendingDestroyComponent = NewComponent[AnchorPendingDestroy]()
