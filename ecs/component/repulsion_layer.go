package component

// RepulsionLayer allows entities to declare a repulsion category and mask
// so the cluster repulsion system can selectively enable/disable repulsion
// between groups of objects.
type RepulsionLayer struct {
	// Category is a bitmask of this entity's repulsion category. If zero,
	// the repulsion system treats it as category 1.
	Category uint32 `json:"category,omitempty"`
	// Mask is a bitmask of categories this entity should repel with. If zero,
	// the repulsion system treats it as all-bits set (repel with all).
	Mask uint32 `json:"mask,omitempty"`
}

var RepulsionLayerComponent = NewComponent[RepulsionLayer]()
