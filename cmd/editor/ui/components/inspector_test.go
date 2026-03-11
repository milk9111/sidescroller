package components

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
)

func TestInspectorSyncDoesNotEmitFieldEditsDuringRebuild(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editCount := 0
	panel := NewInspectorPanel(theme, func(InspectorFieldEdit) {
		editCount++
	})

	panel.Sync(InspectorState{
		Active:      true,
		EntityLabel: "player",
		PrefabPath:  "player.yaml",
		Sections: []InspectorSectionState{{
			Component: "animation",
			Label:     "Animation",
			Fields: []InspectorFieldState{{
				Component: "animation",
				Field:     "defs",
				Label:     "Defs",
				TypeLabel: "yaml[]",
				Value:     "- name: idle\n  row: 0\n  frame_count: 5",
			}},
		}},
	})

	if editCount != 0 {
		t.Fatalf("expected sync rebuild to avoid synthetic edits, got %d", editCount)
	}
}

func TestInspectorStructureKeyIgnoresSectionAndFieldOrder(t *testing.T) {
	first := InspectorState{
		Active:      true,
		EntityLabel: "player",
		Sections: []InspectorSectionState{
			{
				Component: "transform",
				Label:     "Transform",
				Fields: []InspectorFieldState{
					{Component: "transform", Field: "y", Label: "Y", TypeLabel: "float"},
					{Component: "transform", Field: "x", Label: "X", TypeLabel: "float"},
				},
			},
			{
				Component: "sprite",
				Label:     "Sprite",
				Fields: []InspectorFieldState{
					{Component: "sprite", Field: "image", Label: "Image", TypeLabel: "string"},
				},
			},
		},
	}

	second := InspectorState{
		Active:      true,
		EntityLabel: "player",
		Sections: []InspectorSectionState{
			{
				Component: "sprite",
				Label:     "Sprite",
				Fields: []InspectorFieldState{
					{Component: "sprite", Field: "image", Label: "Image", TypeLabel: "string"},
				},
			},
			{
				Component: "transform",
				Label:     "Transform",
				Fields: []InspectorFieldState{
					{Component: "transform", Field: "x", Label: "X", TypeLabel: "float"},
					{Component: "transform", Field: "y", Label: "Y", TypeLabel: "float"},
				},
			},
		},
	}

	if got, want := inspectorStructureKey(first), inspectorStructureKey(second); got != want {
		t.Fatalf("expected identical structure keys for reordered content, got %q != %q", got, want)
	}
}

func TestInspectorFocusedInputReturnsFocusedField(t *testing.T) {
	first := widget.NewTextInput()
	second := widget.NewTextInput()
	second.Focus(true)

	panel := &InspectorPanel{
		inputs: map[string]*widget.TextInput{
			"a": first,
			"b": second,
		},
		currentKeyList: []string{"a", "b"},
	}

	if got := panel.FocusedInput(); got != second {
		t.Fatalf("expected focused input %p, got %p", second, got)
	}
	if !panel.AnyInputFocused() {
		t.Fatalf("expected AnyInputFocused to report true")
	}
}

func TestInspectorSyncTogglesPrebuiltSectionVisibility(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	state := InspectorState{
		Sections: []InspectorSectionState{
			{
				Component: "transform",
				Label:     "Transform",
				Fields: []InspectorFieldState{{
					Component: "transform",
					Field:     "x",
					Label:     "X",
					TypeLabel: "float64",
				}},
			},
			{
				Component: "sprite",
				Label:     "Sprite",
				Fields: []InspectorFieldState{{
					Component: "sprite",
					Field:     "image",
					Label:     "Image",
					TypeLabel: "string",
				}},
			},
		},
	}

	panel.Sync(state)
	if panel.sections["transform"].Root.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected transform section to be hidden before selection")
	}
	if panel.sections["sprite"].Root.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected sprite section to be hidden before selection")
	}

	state.Active = true
	state.EntityLabel = "player"
	state.Sections[0].Visible = true
	state.Sections[0].Fields[0].Value = "42"
	panel.Sync(state)

	if panel.sections["transform"].Root.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected transform section to be shown for selected entity")
	}
	if panel.sections["sprite"].Root.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected unrelated sprite section to remain hidden")
	}
	if got := panel.inputs[inspectorFieldKey("transform", "x")].GetText(); got != "42" {
		t.Fatalf("expected transform.x input to update to selected value, got %q", got)
	}
	if panel.inputs[inspectorFieldKey("sprite", "image")] == nil {
		t.Fatal("expected hidden sprite input to remain constructed")
	}
}
