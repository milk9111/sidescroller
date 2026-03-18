package components

import (
	"image"
	"testing"
	"time"
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

func TestTextEditorDoubleClickSelectsWord(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform: value_here")
	editor.SetLocation(image.Rect(0, 0, 320, 120))
	clickedAt := time.Unix(100, 0)
	clickX := editor.padding + editor.lineMetrics[0].prefixWidths[len([]rune("transform: val"))]
	clickY := editor.padding + (editor.lineHeight / 2)

	editor.handlePrimaryClickAt(clickX, clickY, clickedAt)
	editor.handlePrimaryClickAt(clickX, clickY, clickedAt.Add(200*time.Millisecond))

	start, end, ok := editor.selectionBounds()
	if !ok {
		t.Fatal("expected double click to create a selection")
	}
	if start.row != 0 || start.column != len([]rune("transform: ")) {
		t.Fatalf("expected selection to start at the clicked word, got row=%d col=%d", start.row, start.column)
	}
	if end.row != 0 || end.column != len([]rune("transform: value_here")) {
		t.Fatalf("expected selection to extend to the end of the word, got row=%d col=%d", end.row, end.column)
	}
	if editor.cursorRow != end.row || editor.cursorColumn != end.column {
		t.Fatalf("expected caret at selection end, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}

func TestTextEditorHandlePointerPressFocusesAndPlacesCaret(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform:\n  x: 12")
	editor.SetLocation(image.Rect(32, 48, 352, 208))
	clickedAt := time.Unix(200, 0)
	clickX := editor.GetWidget().Rect.Min.X + editor.padding + editor.lineMetrics[1].prefixWidths[len([]rune("  x:"))]
	clickY := editor.GetWidget().Rect.Min.Y + editor.padding + editor.lineHeight + (editor.lineHeight / 2)

	editor.handlePointerPress(image.Pt(clickX, clickY), true, clickedAt)

	if !editor.IsFocused() {
		t.Fatal("expected inside click to focus the text editor")
	}
	if editor.cursorRow != 1 {
		t.Fatalf("expected caret row 1 after click, got %d", editor.cursorRow)
	}
	if editor.cursorColumn != len([]rune("  x:")) {
		t.Fatalf("expected caret to move near clicked column, got %d", editor.cursorColumn)
	}
	if editor.lastClickAt != clickedAt {
		t.Fatalf("expected click timestamp to be recorded, got %v", editor.lastClickAt)
	}
}

func TestTextEditorHandlePointerPressBlurOnOutsideClick(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetLocation(image.Rect(32, 48, 352, 208))
	editor.Focus(true)

	editor.handlePointerPress(image.Pt(8, 8), true, time.Unix(300, 0))

	if editor.IsFocused() {
		t.Fatal("expected outside click to blur the text editor")
	}
}

func TestTextEditorInsertTextReplacesSelection(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("transform: value_here")
	editor.setSelection(
		textEditorPosition{row: 0, column: len([]rune("transform: "))},
		textEditorPosition{row: 0, column: len([]rune("transform: value_here"))},
	)
	editor.cursorRow = 0
	editor.cursorColumn = len([]rune("transform: value_here"))

	editor.insertText("other")

	if got := editor.GetText(); got != "transform: other" {
		t.Fatalf("expected selected word to be replaced, got %q", got)
	}
	if _, _, ok := editor.selectionBounds(); ok {
		t.Fatal("expected selection to clear after replacement")
	}
	if editor.cursorColumn != len([]rune("transform: other")) {
		t.Fatalf("expected caret after replacement text, got %d", editor.cursorColumn)
	}
}

func TestTextEditorBackspaceDeletesSelection(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	editor := NewTextEditor(theme, nil, nil)
	editor.SetText("alpha beta")
	editor.setSelection(textEditorPosition{row: 0, column: 6}, textEditorPosition{row: 0, column: 10})
	editor.cursorRow = 0
	editor.cursorColumn = 10

	editor.backspace()

	if got := editor.GetText(); got != "alpha " {
		t.Fatalf("expected backspace to delete the selected word, got %q", got)
	}
	if editor.cursorRow != 0 || editor.cursorColumn != 6 {
		t.Fatalf("expected caret to collapse to selection start, got row=%d col=%d", editor.cursorRow, editor.cursorColumn)
	}
}
