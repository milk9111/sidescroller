package system

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
)

func TestTutorialSystemHidesExpiredTutorial(t *testing.T) {
	w := ecs.NewWorld()
	if _, err := levelentity.NewUIRoot(w); err != nil {
		t.Fatalf("new ui root: %v", err)
	}
	if err := levelentity.ShowTutorial(w, "test tutorial", 2); err != nil {
		t.Fatalf("show tutorial: %v", err)
	}

	system := NewTutorialSystem()
	system.Update(w)

	ent, ok := ecs.First(w, component.TutorialStateComponent.Kind())
	if !ok {
		t.Fatal("expected tutorial state entity")
	}
	state, _ := ecs.Get(w, ent, component.TutorialStateComponent.Kind())
	if state == nil || state.RemainingFrames != 1 {
		t.Fatalf("expected one frame remaining, got %+v", state)
	}

	system.Update(w)

	state, _ = ecs.Get(w, ent, component.TutorialStateComponent.Kind())
	if state == nil || state.Active || state.RemainingFrames != 0 {
		t.Fatalf("expected cleared tutorial state, got %+v", state)
	}
	ui, _ := ecs.Get(w, ent, component.TutorialUIComponent.Kind())
	if ui == nil || ui.Text == nil || ui.Text.Label != "" {
		t.Fatalf("expected cleared tutorial ui, got %+v", ui)
	}
	if ui.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected tutorial overlay to be hidden")
	}
}
