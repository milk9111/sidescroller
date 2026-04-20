package entity

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestShowTutorialUpdatesTutorialUIState(t *testing.T) {
	w := ecs.NewWorld()
	if _, err := NewUIRoot(w); err != nil {
		t.Fatalf("new ui root: %v", err)
	}

	if err := ShowTutorial(w, "Wrapped tutorial text", 120); err != nil {
		t.Fatalf("show tutorial: %v", err)
	}

	ent, ok := ecs.First(w, component.TutorialStateComponent.Kind())
	if !ok {
		t.Fatal("expected tutorial state entity")
	}

	state, _ := ecs.Get(w, ent, component.TutorialStateComponent.Kind())
	if state == nil || !state.Active || state.RemainingFrames != 120 {
		t.Fatalf("unexpected tutorial state: %+v", state)
	}

	ui, _ := ecs.Get(w, ent, component.TutorialUIComponent.Kind())
	if ui == nil || ui.Text == nil || ui.Text.Label != "Wrapped tutorial text" {
		t.Fatalf("unexpected tutorial ui: %+v", ui)
	}
	if ui.Overlay.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected tutorial overlay to be visible")
	}
}
