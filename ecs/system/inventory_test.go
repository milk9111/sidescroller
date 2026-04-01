package system

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
)

func TestPlayerPrefabStartsWithWrenchInventory(t *testing.T) {
	w := ecs.NewWorld()

	_, err := entity.BuildEntity(w, "player.yaml")
	if err != nil {
		t.Fatalf("build player prefab: %v", err)
	}

	inventory := currentPlayerInventory(w)
	if inventory == nil {
		t.Fatal("expected inventory component")
	}
	if len(inventory.Items) != 1 {
		t.Fatalf("expected one starting inventory item, got %+v", inventory.Items)
	}
	wrench := inventory.Items[0]
	if wrench.Prefab != "item_wrench.yaml" {
		t.Fatalf("expected starting item prefab item_wrench.yaml, got %q", wrench.Prefab)
	}
	if wrench.Count != 1 {
		t.Fatalf("expected wrench count 1, got %d", wrench.Count)
	}
	definition, err := resolveInventoryItemDefinition(wrench.Prefab)
	if err != nil {
		t.Fatalf("resolve wrench item prefab: %v", err)
	}
	if definition == nil || definition.Image == nil {
		t.Fatal("expected wrench item prefab to resolve to an icon")
	}
}

func TestInventorySystemNavigatesAndCloses(t *testing.T) {
	w := ecs.NewWorld()

	uiEnt, err := entity.NewUIRoot(w)
	if err != nil {
		t.Fatalf("new ui root: %v", err)
	}

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.InventoryComponent.Kind(), &component.Inventory{Items: []component.InventoryItem{{Prefab: "item_wrench.yaml", Count: 1}, {Prefab: "item_gear.yaml", Count: 2}}}); err != nil {
		t.Fatalf("add inventory: %v", err)
	}
	if err := ecs.Add(w, player, component.InputComponent.Kind(), &component.Input{}); err != nil {
		t.Fatalf("add input: %v", err)
	}

	state, ok := ecs.Get(w, uiEnt, component.InventoryStateComponent.Kind())
	if !ok || state == nil {
		t.Fatal("expected inventory state")
	}
	state.Active = true
	if err := ecs.Add(w, uiEnt, component.InventoryStateComponent.Kind(), state); err != nil {
		t.Fatalf("activate inventory: %v", err)
	}

	ui, ok := ecs.Get(w, uiEnt, component.InventoryUIComponent.Kind())
	if !ok || ui == nil {
		t.Fatal("expected inventory ui")
	}

	system := NewInventorySystem()
	system.Update(w)

	if ui.Overlay.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected inventory overlay to be visible")
	}
	if state.SelectedIndex != 0 {
		t.Fatalf("expected initial selection to stay on first item, got %d", state.SelectedIndex)
	}
	if ui.DetailText.Label == "" {
		t.Fatal("expected detail panel text for selected item")
	}

	input, _ := ecs.Get(w, player, component.InputComponent.Kind())
	input.MoveX = 1
	system.Update(w)

	if state.SelectedIndex != 1 {
		t.Fatalf("expected selection to move right to second item, got %d", state.SelectedIndex)
	}
	if ui.DetailText.Label != "A salvaged machine gear. Collect these to raise your gear count." {
		t.Fatalf("expected gear description after navigation, got %q", ui.DetailText.Label)
	}

	input.MoveX = 0
	input.MenuPressed = true
	system.Update(w)

	if state.Active {
		t.Fatal("expected inventory to close on menu press")
	}
	if ui.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected inventory overlay to be hidden after close")
	}
}
