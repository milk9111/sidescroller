package components

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
)

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
