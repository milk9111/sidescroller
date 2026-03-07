package system

import (
	"sort"
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestDrawLayerIndexPrefersEntityLayer(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 7})
	_ = ecs.Add(w, e, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 2})

	if got := drawLayerIndex(w, e); got != 2 {
		t.Fatalf("expected draw layer 2, got %d", got)
	}
	if got := renderOrderIndex(w, e); got != 7 {
		t.Fatalf("expected render order 7, got %d", got)
	}
}

func TestEntityLayerScopesRenderOrder(t *testing.T) {
	w := ecs.NewWorld()
	backHighOrder := ecs.CreateEntity(w)
	frontLowOrder := ecs.CreateEntity(w)
	for _, tc := range []struct {
		entity     ecs.Entity
		layerIndex int
		orderIndex int
	}{
		{entity: backHighOrder, layerIndex: 0, orderIndex: 99},
		{entity: frontLowOrder, layerIndex: 1, orderIndex: 0},
	} {
		_ = ecs.Add(w, tc.entity, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: tc.layerIndex})
		_ = ecs.Add(w, tc.entity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: tc.orderIndex})
	}

	entities := []ecs.Entity{frontLowOrder, backHighOrder}
	sort.Slice(entities, func(i, j int) bool {
		li := drawLayerIndex(w, entities[i])
		lj := drawLayerIndex(w, entities[j])
		if li != lj {
			return li < lj
		}
		oi := renderOrderIndex(w, entities[i])
		oj := renderOrderIndex(w, entities[j])
		if oi != oj {
			return oi < oj
		}
		return uint64(entities[i]) < uint64(entities[j])
	})

	if entities[0] != backHighOrder {
		t.Fatalf("expected lower entity layer to sort first regardless of render order")
	}
}
