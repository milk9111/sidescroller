package components

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
)

func TestPanelTextHelpersClampWidth(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	valueText := newValueText(theme)
	if valueText.MaxWidth != scrollableListMaxWidth {
		t.Fatalf("expected value text max width %v, got %v", scrollableListMaxWidth, valueText.MaxWidth)
	}

	layoutData, ok := valueText.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatalf("expected RowLayoutData for value text, got %T", valueText.GetWidget().LayoutData)
	}
	if !layoutData.Stretch || layoutData.MaxWidth != scrollableListMaxWidth {
		t.Fatalf("expected stretched value text capped at %v, got %+v", scrollableListMaxWidth, layoutData)
	}
}

func TestPanelTextInputsClampWidth(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	input := newEditorTextInput(theme, nil)
	layoutData, ok := input.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatalf("expected RowLayoutData for text input, got %T", input.GetWidget().LayoutData)
	}
	if !layoutData.Stretch || layoutData.MaxWidth != scrollableListMaxWidth {
		t.Fatalf("expected stretched input capped at %v, got %+v", scrollableListMaxWidth, layoutData)
	}
}

func TestTransitionPanelDraftStateUsesLiveInputValues(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	panel := NewTransitionPanel(theme, LayerCallbacks{})
	panel.currentState = TransitionEditorState{Selected: true, ID: "t1", ToLevel: "zone_a", LinkedID: "right", EnterDir: "down"}
	panel.IDInput.SetText("t1")
	panel.ToLevelInput.SetText("zone_b")
	panel.LinkedInput.SetText("upper_right")

	state, ok := panel.DraftState()
	if !ok {
		t.Fatal("expected draft state to be available for selected transition")
	}
	if state.LinkedID != "upper_right" {
		t.Fatalf("expected live linked_id upper_right, got %q", state.LinkedID)
	}
	if state.ToLevel != "zone_b" {
		t.Fatalf("expected live to_level zone_b, got %q", state.ToLevel)
	}
	if state.EnterDir != "down" {
		t.Fatalf("expected enter_dir down, got %q", state.EnterDir)
	}
}

func TestTransitionPanelSyncPreservesLocalDraftWhenWorldStateIsStale(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	panel := NewTransitionPanel(theme, LayerCallbacks{})
	panel.Sync(true, nil, 7, TransitionEditorState{Selected: true, ID: "t1", ToLevel: "zone_a", LinkedID: "right", EnterDir: "down"})
	panel.LinkedInput.SetText("upper_right")
	panel.currentState.LinkedID = "upper_right"
	panel.currentState.Selected = true
	panel.draftDirty = true

	panel.Sync(true, nil, 7, TransitionEditorState{Selected: true, ID: "t1", ToLevel: "zone_a", LinkedID: "right", EnterDir: "down"})
	state, ok := panel.DraftState()
	if !ok {
		t.Fatal("expected draft state after sync")
	}
	if state.LinkedID != "upper_right" {
		t.Fatalf("expected local draft linked_id upper_right, got %q", state.LinkedID)
	}
	if !panel.draftDirty {
		t.Fatal("expected draftDirty to remain set while world state is stale")
	}
}
