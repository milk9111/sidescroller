package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
)

func TestTutorialModuleShowUpdatesTutorialUI(t *testing.T) {
	w := ecs.NewWorld()
	if _, err := levelentity.NewUIRoot(w); err != nil {
		t.Fatalf("new ui root: %v", err)
	}

	mod := TutorialModule().Build(w, nil, 0, 0)
	showFn, ok := mod["show"].(*tengo.UserFunction)
	if !ok || showFn == nil {
		t.Fatal("expected tutorial show function")
	}

	result, err := showFn.Value(&tengo.String{Value: "Press C to use the healing flask."}, &tengo.Int{Value: 180})
	if err != nil {
		t.Fatalf("tutorial show: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("expected true result, got %#v", result)
	}

	ent, ok := ecs.First(w, component.TutorialStateComponent.Kind())
	if !ok {
		t.Fatal("expected tutorial state entity")
	}
	state, _ := ecs.Get(w, ent, component.TutorialStateComponent.Kind())
	if state == nil || !state.Active || state.RemainingFrames != 180 {
		t.Fatalf("unexpected tutorial state: %+v", state)
	}
	ui, _ := ecs.Get(w, ent, component.TutorialUIComponent.Kind())
	if ui == nil || ui.Text == nil || ui.Text.Label != "Press C to use the healing flask." {
		t.Fatalf("unexpected tutorial ui: %+v", ui)
	}
	if ui.Overlay.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected tutorial overlay to be visible")
	}
}
