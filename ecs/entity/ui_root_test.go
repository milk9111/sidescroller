package entity

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestNewUIRootCentersItemOverlayContent(t *testing.T) {
	w := ecs.NewWorld()

	ent, err := NewUIRoot(w)
	if err != nil {
		t.Fatalf("new ui root: %v", err)
	}

	itemUI, ok := ecs.Get(w, ent, component.ItemUIComponent.Kind())
	if !ok || itemUI == nil {
		t.Fatal("expected item ui component")
	}

	imageLayout, ok := itemUI.Image.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatal("expected item image row layout data")
	}
	if imageLayout.Position != widget.RowLayoutPositionCenter {
		t.Fatalf("expected item image to be centered, got %v", imageLayout.Position)
	}

	textLayout, ok := itemUI.Text.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatal("expected item text row layout data")
	}
	if textLayout.Position != widget.RowLayoutPositionCenter {
		t.Fatalf("expected item text block to be centered, got %v", textLayout.Position)
	}
	if !textLayout.Stretch {
		t.Fatal("expected item text block to stretch to panel width")
	}
}

func TestNewUIRootAddsInventoryOverlay(t *testing.T) {
	w := ecs.NewWorld()

	ent, err := NewUIRoot(w)
	if err != nil {
		t.Fatalf("new ui root: %v", err)
	}

	inventoryUI, ok := ecs.Get(w, ent, component.InventoryUIComponent.Kind())
	if !ok || inventoryUI == nil {
		t.Fatal("expected inventory ui component")
	}
	if inventoryUI.Overlay == nil || inventoryUI.GridHost == nil || inventoryUI.DetailImage == nil || inventoryUI.DetailText == nil {
		t.Fatal("expected inventory overlay widgets to be initialized")
	}
	dialogueUI, ok := ecs.Get(w, ent, component.DialogueUIComponent.Kind())
	if !ok || dialogueUI == nil || dialogueUI.HUDLayer == nil || dialogueUI.OverlayLayer == nil {
		t.Fatal("expected dialogue ui layering containers to be initialized")
	}
	if inventoryUI.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected inventory overlay to start hidden")
	}

	layout, ok := inventoryUI.Panel.GetWidget().LayoutData.(widget.AnchorLayoutData)
	if !ok {
		t.Fatal("expected inventory panel anchor layout data")
	}
	if !layout.StretchHorizontal || !layout.StretchVertical {
		t.Fatal("expected inventory panel to stretch across the screen")
	}

	gridLayout, ok := inventoryUI.GridHost.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatal("expected inventory grid host row layout data")
	}
	if !gridLayout.Stretch {
		t.Fatal("expected inventory grid host to stretch within its panel")
	}
	if inventoryUI.GridHost.GetWidget().MinWidth <= 0 || inventoryUI.GridHost.GetWidget().MinHeight <= 0 {
		t.Fatal("expected inventory grid host to have minimum size")
	}
	if inventoryUI.GridHost.GetWidget().MinWidth <= inventoryUI.DetailPanel.GetWidget().MinWidth {
		t.Fatalf("expected inventory grid side to be wider than detail side, got %d vs %d", inventoryUI.GridHost.GetWidget().MinWidth, inventoryUI.DetailPanel.GetWidget().MinWidth)
	}
	if inventoryUI.DetailPanel.GetWidget().MinHeight != inventoryBodyMinHeight {
		t.Fatalf("expected inventory detail panel to use full remaining height, got %d want %d", inventoryUI.DetailPanel.GetWidget().MinHeight, inventoryBodyMinHeight)
	}

	tutorialUI, ok := ecs.Get(w, ent, component.TutorialUIComponent.Kind())
	if !ok || tutorialUI == nil {
		t.Fatal("expected tutorial ui component")
	}
	if tutorialUI.Overlay == nil || tutorialUI.Panel == nil || tutorialUI.Text == nil {
		t.Fatal("expected tutorial ui widgets to be initialized")
	}
	if tutorialUI.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected tutorial overlay to start hidden")
	}
}
