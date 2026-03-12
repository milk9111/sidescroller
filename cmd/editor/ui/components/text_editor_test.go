package components

import (
	"image"
	"testing"
)

func TestTextEditorSetTextNormalizesNewlines(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform:\r\n  x: 1\r\n  y: 2")

	if got := editor.GetText(); got != "transform:\n  x: 1\n  y: 2" {
		t.Fatalf("expected normalized text, got %q", got)
	}
	if editor.IsDirty() {
		t.Fatal("expected SetText to clear the dirty flag")
	}
}

func TestTextEditorInsertNewlineCopiesIndentAndColonOffset(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform:")
	editor.cursorRow = 0
	editor.cursorColumn = len([]rune("transform:"))

	editor.insertNewlineWithIndent()

	if got := editor.GetText(); got != "transform:\n  " {
		t.Fatalf("expected newline with soft-tab indent, got %q", got)
	}
	if editor.cursorRow != 1 || editor.cursorColumn != 2 {
		t.Fatalf("expected caret on indented next line, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}

func TestTextEditorInsertNewlineKeepsIndentOnBlankIndentedLine(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("    ")
	editor.cursorRow = 0
	editor.cursorColumn = 2

	editor.insertNewlineWithIndent()

	if got := editor.GetText(); got != "  \n    " {
		t.Fatalf("expected blank indented line to preserve indent level without duplicating trailing spaces, got %q", got)
	}
	if editor.cursorRow != 1 || editor.cursorColumn != 4 {
		t.Fatalf("expected caret on preserved indent, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}

func TestTextEditorBackspaceJoinsLinesAtColumnZero(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform:\n  x: 1")
	editor.cursorRow = 1
	editor.cursorColumn = 0

	editor.backspace()

	if got := editor.GetText(); got != "transform:  x: 1" {
		t.Fatalf("expected joined lines after backspace, got %q", got)
	}
	if editor.cursorRow != 0 || editor.cursorColumn != len([]rune("transform:")) {
		t.Fatalf("expected caret at join point, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}

func TestTextEditorOutdentRemovesSoftTab(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("  x: 1")
	editor.cursorRow = 0
	editor.cursorColumn = 2

	editor.outdentCurrentLine()

	if got := editor.GetText(); got != "x: 1" {
		t.Fatalf("expected outdented line, got %q", got)
	}
	if editor.cursorColumn != 0 {
		t.Fatalf("expected caret to move with removed indent, got %d", editor.cursorColumn)
	}
}

func TestTextEditorDeleteForwardJoinsWithNextLineAtEndOfLine(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform:\n  x: 1")
	editor.cursorRow = 0
	editor.cursorColumn = len([]rune("transform:"))

	editor.deleteForward()

	if got := editor.GetText(); got != "transform:  x: 1" {
		t.Fatalf("expected delete at end-of-line to join the next line, got %q", got)
	}
	if editor.cursorRow != 0 || editor.cursorColumn != len([]rune("transform:")) {
		t.Fatalf("expected caret to stay at join point, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}

func TestTextEditorMoveCursorVerticalPreservesDesiredColumn(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("abcdef\nxy\n123456")
	editor.cursorRow = 0
	editor.cursorColumn = 5

	editor.moveCursorVertical(1)
	if editor.cursorRow != 1 || editor.cursorColumn != 2 {
		t.Fatalf("expected down movement to clamp to short line, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
	editor.moveCursorVertical(1)
	if editor.cursorRow != 2 || editor.cursorColumn != 5 {
		t.Fatalf("expected down movement to restore desired column on longer line, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}

func TestTextEditorRebuildLineMetricsCachesWidthsAndBlankState(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform:\n  x: 1")

	if len(editor.lineMetrics) != 2 {
		t.Fatalf("expected 2 cached line metrics, got %d", len(editor.lineMetrics))
	}
	if editor.blankDocument {
		t.Fatal("expected non-blank document cache state")
	}
	if editor.maxLineWidth <= 0 {
		t.Fatalf("expected cached max line width > 0, got %d", editor.maxLineWidth)
	}
	if got := editor.lineMetrics[1].prefixWidths[len(editor.lineMetrics[1].prefixWidths)-1]; got != editor.lineMetrics[1].width {
		t.Fatalf("expected final prefix width to match cached width, got prefix=%d width=%d", got, editor.lineMetrics[1].width)
	}
}

func TestTextEditorSetScrollOffsetsClampsAndReportsChange(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("alpha\nbeta\ngamma\ndelta\nepsilon\nzeta")
	editor.SetLocation(image.Rect(0, 0, 120, 64))

	if changed := editor.setScrollOffsets(9999, 9999); !changed {
		t.Fatal("expected large scroll request to clamp and report a change")
	}
	if editor.scrollY == 0 {
		t.Fatal("expected vertical scroll to clamp above zero")
	}
	if changed := editor.setScrollOffsets(editor.scrollX, editor.scrollY); changed {
		t.Fatal("expected identical scroll offsets to report no change")
	}
}

func TestTextEditorHandleMouseWheelScrollsWhenHoveredWithoutFocus(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
	editor.SetLocation(image.Rect(0, 0, 120, 64))

	if handled := editor.handleMouseWheel(image.Pt(20, 20), -1); !handled {
		t.Fatal("expected hovered mouse wheel input to be handled")
	}
	if editor.scrollY == 0 {
		t.Fatal("expected hovered mouse wheel input to move the vertical scroll position")
	}
}

func TestTextEditorHandleMouseWheelIgnoresPointerOutsideWidget(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("a\nb\nc\nd\ne\nf\ng\nh\ni\nj")
	editor.SetLocation(image.Rect(0, 0, 120, 64))

	if handled := editor.handleMouseWheel(image.Pt(180, 20), -1); handled {
		t.Fatal("expected mouse wheel input outside the widget to be ignored")
	}
	if editor.scrollY != 0 {
		t.Fatalf("expected outside mouse wheel input not to scroll, got %d", editor.scrollY)
	}
}

func TestTextEditorSetMinHeightUpdatesWidgetMinimum(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)

	if changed := editor.SetMinHeight(640); !changed {
		t.Fatal("expected SetMinHeight to report a changed minimum height")
	}
	if editor.GetWidget().MinHeight != 640 {
		t.Fatalf("expected widget minimum height 640, got %d", editor.GetWidget().MinHeight)
	}
}

func TestTextEditorFocusedWheelScrollCanMovePastCaretLine(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("0\n1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14")
	editor.SetLocation(image.Rect(0, 0, 160, 96))
	editor.Focus(true)
	editor.cursorRow = 0
	editor.cursorColumn = 0

	for index := 0; index < 6; index++ {
		editor.handleMouseWheel(image.Pt(20, 20), -1)
	}

	if editor.scrollY <= editor.lineHeight {
		t.Fatalf("expected wheel scrolling to move well past the caret line, got scrollY=%d lineHeight=%d", editor.scrollY, editor.lineHeight)
	}
	if editor.cursorRow != 0 || editor.cursorColumn != 0 {
		t.Fatalf("expected scrolling not to move the caret, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}
