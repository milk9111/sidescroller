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