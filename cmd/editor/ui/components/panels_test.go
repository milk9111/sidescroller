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
