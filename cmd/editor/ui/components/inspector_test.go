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
