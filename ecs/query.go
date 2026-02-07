package ecs

// IntersectEntities returns entity IDs present in both sets.
func IntersectEntities(a, b *SparseSet) []int {
	if a == nil || b == nil {
		return nil
	}
	// iterate smaller set
	if len(a.denseEntities) > len(b.denseEntities) {
		a, b = b, a
	}
	out := make([]int, 0, len(a.denseEntities))
	for _, id := range a.denseEntities {
		if b.Has(id) {
			out = append(out, id)
		}
	}
	return out
}
