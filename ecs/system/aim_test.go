package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestCopyAnchorDrawOrderPlacesAnchorJustBehindPlayer(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	anchor := ecs.CreateEntity(w)

	_ = ecs.Add(w, player, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 4})
	_ = ecs.Add(w, player, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 100})
	_ = ecs.Add(w, anchor, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 56})

	copyAnchorDrawOrder(w, player, anchor)

	anchorLayer, ok := ecs.Get(w, anchor, component.EntityLayerComponent.Kind())
	if !ok || anchorLayer == nil {
		t.Fatal("expected anchor entity layer to be copied from player")
	}
	if anchorLayer.Index != 4 {
		t.Fatalf("expected anchor entity layer 4, got %d", anchorLayer.Index)
	}

	anchorOrder, ok := ecs.Get(w, anchor, component.RenderLayerComponent.Kind())
	if !ok || anchorOrder == nil {
		t.Fatal("expected anchor render order to be set")
	}
	if anchorOrder.Index != 99 {
		t.Fatalf("expected anchor render order 99, got %d", anchorOrder.Index)
	}

	if got := drawLayerIndex(w, anchor); got != 4 {
		t.Fatalf("expected anchor draw layer 4, got %d", got)
	}
	if got := renderOrderIndex(w, anchor); got != 99 {
		t.Fatalf("expected anchor render order index 99, got %d", got)
	}
}
