package components

import (
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	textEditorMinHeight    = 320
	textEditorPadding      = 8
	textEditorSoftTab      = "  "
	textEditorRepeatDelay  = 15
	textEditorRepeatPeriod = 4
	textEditorBlinkPeriod  = 1060 * time.Millisecond
	textEditorCaretWidth   = 2
)

var textEditorSolidPixel = ebiten.NewImage(1, 1)

func init() {
	textEditorSolidPixel.Fill(color.White)
}

type TextEditor struct {
	theme           *Theme
	widget          *widget.Widget
	focusMap        map[widget.FocusDirection]widget.Focuser
	onChanged       func(string)
	onSaveRequested func(string)

	focused       bool
	tabOrder      int
	documentText  string
	lines         []string
	cursorRow     int
	cursorColumn  int
	desiredColumn int
	dirty         bool

	padding       int
	softTab       string
	lineHeight    int
	scrollX       int
	scrollY       int
	lineMetrics   []textEditorLineMetrics
	maxLineWidth  int
	contentHeight int
	blankDocument bool

	viewport      *ebiten.Image
	viewportSize  image.Point
	renderDirty   bool
	lastCaretDraw bool
	blinkResetAt  time.Time

	backgroundIdle     *euiimage.NineSlice
	backgroundDisabled *euiimage.NineSlice
	textColor          color.Color
	mutedTextColor     color.Color
}

type textEditorLineMetrics struct {
	text         string
	runes        []rune
	prefixWidths []int
	width        int
}

func NewTextEditor(theme *Theme, onChanged func(string), onSaveRequested func(string), widgetOpts ...widget.WidgetOpt) *TextEditor {
	editor := &TextEditor{
		theme:              theme,
		focusMap:           make(map[widget.FocusDirection]widget.Focuser),
		onChanged:          onChanged,
		onSaveRequested:    onSaveRequested,
		tabOrder:           -1,
		padding:            textEditorPadding,
		softTab:            textEditorSoftTab,
		desiredColumn:      -1,
		renderDirty:        true,
		blinkResetAt:       time.Now(),
		backgroundIdle:     theme.InputImage.Idle,
		backgroundDisabled: theme.InputImage.Disabled,
		textColor:          theme.InputColor.Idle,
		mutedTextColor:     theme.MutedTextColor,
	}
	editor.widget = widget.NewWidget(append([]widget.WidgetOpt{
		widget.WidgetOpts.TrackHover(true),
		widget.WidgetOpts.MinSize(scrollableListMaxWidth, textEditorMinHeight),
	}, widgetOpts...)...)
	editor.widget.MouseButtonPressedEvent.AddHandler(func(args any) {
		eventArgs, ok := args.(*widget.WidgetMouseButtonPressedEventArgs)
		if !ok || eventArgs.Button != ebiten.MouseButtonLeft {
			return
		}
		editor.Focus(true)
		editor.setCaretFromOffset(eventArgs.OffsetX, eventArgs.OffsetY)
	})
	editor.setTextInternal("", false, false)
	editor.Validate()
	return editor
}

func (t *TextEditor) GetWidget() *widget.Widget {
	return t.widget
}

func (t *TextEditor) PreferredSize() (int, int) {
	if t == nil || t.widget == nil {
		return scrollableListMaxWidth, textEditorMinHeight
	}
	width := t.widget.MinWidth
	if width <= 0 {
		width = scrollableListMaxWidth
	}
	height := t.widget.MinHeight
	if height <= 0 {
		height = textEditorMinHeight
	}
	return width, height
}

func (t *TextEditor) SetLocation(rect image.Rectangle) {
	if t == nil || t.widget == nil {
		return
	}
	if t.widget.Rect == rect {
		return
	}
	t.widget.Rect = rect
	t.renderDirty = true
	t.clampScroll()
	if rect.Dx() != t.viewportSize.X || rect.Dy() != t.viewportSize.Y {
		t.viewport = nil
		t.viewportSize = image.Point{}
	}
	if t.focused {
		t.ensureCaretVisible()
	}
}

func (t *TextEditor) Validate() {
	if t == nil || t.theme == nil {
		return
	}
	_, measuredHeight := textv2.Measure("Mg", t.theme.Face, 0)
	lineHeight := int(math.Ceil(measuredHeight)) + 4
	if lineHeight < 16 {
		lineHeight = 16
	}
	t.lineHeight = lineHeight
	if len(t.lines) == 0 {
		t.lines = []string{""}
	}
	t.rebuildLineMetrics()
	t.clampCursor()
	t.clampScroll()
	t.renderDirty = true
}

func (t *TextEditor) Render(screen *ebiten.Image) {
	if t == nil || t.widget == nil {
		return
	}
	t.widget.Render(screen)
	rect := t.widget.Rect
	if rect.Dx() <= 0 || rect.Dy() <= 0 || t.widget.Visibility != widget.Visibility_Show {
		return
	}
	caretVisible := t.caretVisible()
	if t.viewport == nil || t.viewportSize.X != rect.Dx() || t.viewportSize.Y != rect.Dy() {
		t.viewport = ebiten.NewImage(rect.Dx(), rect.Dy())
		t.viewportSize = image.Pt(rect.Dx(), rect.Dy())
		t.renderDirty = true
	}
	if t.renderDirty || caretVisible != t.lastCaretDraw {
		t.redrawViewport(caretVisible)
		t.lastCaretDraw = caretVisible
		t.renderDirty = false
	}
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y))
	screen.DrawImage(t.viewport, options)
}

func (t *TextEditor) Update(updObj *widget.UpdateObject) {
	if t == nil || t.widget == nil {
		return
	}
	t.widget.Update(updObj)
	cursorX, cursorY := ebiten.CursorPosition()
	if t.focused && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if !pointInRect(t.widget.Rect, image.Pt(cursorX, cursorY)) {
			t.Focus(false)
		}
	}
	_, wheelY := ebiten.Wheel()
	t.handleMouseWheel(image.Pt(cursorX, cursorY), wheelY)
	if t.widget.Visibility != widget.Visibility_Show || t.widget.Disabled || !t.focused {
		return
	}
	if t.handleSaveShortcut() {
		return
	}
	t.handleNavigationKeys()
	t.handleEditingKeys()
	t.handleTextInput()
}

func (t *TextEditor) GetText() string {
	if t == nil {
		return ""
	}
	return t.documentText
}

func (t *TextEditor) SetText(text string) {
	if t == nil {
		return
	}
	t.setTextInternal(text, false, false)
	t.dirty = false
	if !t.focused {
		t.setScrollOffsets(0, 0)
	}
	t.renderDirty = true
}

func (t *TextEditor) SetDirty(dirty bool) {
	if t == nil || t.dirty == dirty {
		return
	}
	t.dirty = dirty
	t.renderDirty = true
}

func (t *TextEditor) SetMinHeight(height int) bool {
	if t == nil || t.widget == nil {
		return false
	}
	if height < textEditorMinHeight {
		height = textEditorMinHeight
	}
	if t.widget.MinHeight == height {
		return false
	}
	t.widget.MinHeight = height
	t.clampScroll()
	t.renderDirty = true
	return true
}

func (t *TextEditor) IsDirty() bool {
	if t == nil {
		return false
	}
	return t.dirty
}

func (t *TextEditor) Focus(focused bool) {
	if t == nil || t.widget == nil || t.focused == focused {
		return
	}
	t.widget.FireFocusEvent(t, focused, image.Point{-1, -1})
	t.focused = focused
	t.resetBlink()
	if focused {
		t.ensureCaretVisible()
	}
	t.renderDirty = true
}

func (t *TextEditor) IsFocused() bool {
	if t == nil {
		return false
	}
	return t.focused
}

func (t *TextEditor) TabOrder() int {
	if t == nil {
		return -1
	}
	return t.tabOrder
}

func (t *TextEditor) GetFocus(direction widget.FocusDirection) widget.Focuser {
	if t == nil {
		return nil
	}
	return t.focusMap[direction]
}

func (t *TextEditor) AddFocus(direction widget.FocusDirection, focus widget.Focuser) {
	if t == nil {
		return
	}
	t.focusMap[direction] = focus
}

func (t *TextEditor) setTextInternal(text string, markDirty bool, notify bool) {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	if normalized == "" {
		t.lines = []string{""}
		t.documentText = ""
	} else {
		t.lines = strings.Split(normalized, "\n")
		t.documentText = normalized
	}
	t.rebuildLineMetrics()
	t.clampCursor()
	t.dirty = markDirty
	t.renderDirty = true
	t.resetBlink()
	if t.focused {
		t.ensureCaretVisible()
	}
	if notify && t.onChanged != nil {
		t.onChanged(t.documentText)
	}
}

func (t *TextEditor) syncDocumentFromLines(notify bool) {
	if len(t.lines) == 0 {
		t.lines = []string{""}
	}
	t.documentText = strings.Join(t.lines, "\n")
	t.rebuildLineMetrics()
	t.dirty = true
	t.renderDirty = true
	t.resetBlink()
	if t.focused {
		t.ensureCaretVisible()
	}
	if notify && t.onChanged != nil {
		t.onChanged(t.documentText)
	}
}

func (t *TextEditor) handleSaveShortcut() bool {
	if !textEditorModifierPressed() || !inpututil.IsKeyJustPressed(ebiten.KeyS) {
		return false
	}
	if t.onSaveRequested != nil {
		t.onSaveRequested(t.GetText())
	}
	return true
}

func (t *TextEditor) handleNavigationKeys() {
	if textEditorKeyTriggered(ebiten.KeyLeft) {
		t.moveCursorLeft()
	}
	if textEditorKeyTriggered(ebiten.KeyRight) {
		t.moveCursorRight()
	}
	if textEditorKeyTriggered(ebiten.KeyUp) {
		t.moveCursorVertical(-1)
	}
	if textEditorKeyTriggered(ebiten.KeyDown) {
		t.moveCursorVertical(1)
	}
	if textEditorKeyTriggered(ebiten.KeyHome) {
		t.moveCursorHome()
	}
	if textEditorKeyTriggered(ebiten.KeyEnd) {
		t.moveCursorEnd()
	}
	if textEditorKeyTriggered(ebiten.KeyPageUp) {
		t.scrollByLines(-t.visibleLineCount())
	}
	if textEditorKeyTriggered(ebiten.KeyPageDown) {
		t.scrollByLines(t.visibleLineCount())
	}
}

func (t *TextEditor) handleEditingKeys() {
	if textEditorModifierPressed() {
		return
	}
	if textEditorKeyTriggered(ebiten.KeyBackspace) {
		t.backspace()
	}
	if textEditorKeyTriggered(ebiten.KeyDelete) {
		t.deleteForward()
	}
	if textEditorKeyTriggered(ebiten.KeyEnter) || textEditorKeyTriggered(ebiten.KeyNumpadEnter) {
		t.insertNewlineWithIndent()
	}
	if textEditorKeyTriggered(ebiten.KeyTab) {
		if textEditorShiftPressed() {
			t.outdentCurrentLine()
		} else {
			t.insertText(t.softTab)
		}
	}
}

func (t *TextEditor) handleTextInput() {
	if textEditorModifierPressed() {
		return
	}
	inputRunes := ebiten.AppendInputChars(nil)
	if len(inputRunes) == 0 {
		return
	}
	filtered := make([]rune, 0, len(inputRunes))
	for _, value := range inputRunes {
		if value < 0x20 || value == 0x7f || value == '\n' || value == '\r' || value == '\t' {
			continue
		}
		filtered = append(filtered, value)
	}
	if len(filtered) == 0 {
		return
	}
	t.insertText(string(filtered))
}

func (t *TextEditor) redrawViewport(caretVisible bool) {
	width := t.viewportSize.X
	height := t.viewportSize.Y
	if width <= 0 || height <= 0 {
		return
	}
	t.viewport.Clear()
	t.drawBackground(width, height)
	innerWidth := maxInt(1, width-(2*t.padding))
	innerHeight := maxInt(1, height-(2*t.padding))
	firstVisibleRow := maxInt(0, t.scrollY/maxInt(1, t.lineHeight))
	lastVisibleRow := minInt(len(t.lines)-1, (t.scrollY+innerHeight)/maxInt(1, t.lineHeight)+1)
	if t.focused && t.cursorRow >= firstVisibleRow && t.cursorRow <= lastVisibleRow {
		lineY := t.padding + (t.cursorRow * t.lineHeight) - t.scrollY
		t.drawSolidRect(t.viewport, t.padding, lineY, innerWidth, t.lineHeight, color.NRGBA{R: 68, G: 114, B: 255, A: 30})
	}
	for rowIndex := firstVisibleRow; rowIndex <= lastVisibleRow && rowIndex < len(t.lines); rowIndex++ {
		metrics := t.lineMetrics[rowIndex]
		var drawOptions textv2.DrawOptions
		drawOptions.GeoM.Translate(float64(t.padding-t.scrollX), float64(t.padding+(rowIndex*t.lineHeight)-t.scrollY))
		drawOptions.ColorScale.ScaleWithColor(t.textColor)
		textv2.Draw(t.viewport, metrics.text, t.theme.Face, &drawOptions)
	}
	if t.focused && caretVisible {
		caretX, caretY := t.caretDrawPosition()
		t.drawSolidRect(t.viewport, caretX, caretY, textEditorCaretWidth, t.lineHeight, t.textColor)
	}
	if t.blankDocument && !t.focused {
		var drawOptions textv2.DrawOptions
		drawOptions.GeoM.Translate(float64(t.padding), float64(t.padding))
		drawOptions.ColorScale.ScaleWithColor(t.mutedTextColor)
		textv2.Draw(t.viewport, "Component YAML", t.theme.Face, &drawOptions)
	}
}

func (t *TextEditor) drawBackground(width, height int) {
	background := t.backgroundIdle
	if t.widget.Disabled && t.backgroundDisabled != nil {
		background = t.backgroundDisabled
	}
	if background != nil {
		background.Draw(t.viewport, width, height, func(*ebiten.DrawImageOptions) {})
	} else {
		t.viewport.Fill(color.NRGBA{R: 42, G: 45, B: 56, A: 255})
	}
	if t.focused {
		t.drawBorder(color.NRGBA{R: 68, G: 114, B: 255, A: 255})
	} else {
		t.drawBorder(color.NRGBA{R: 64, G: 70, B: 86, A: 255})
	}
}

func (t *TextEditor) drawBorder(borderColor color.Color) {
	width := t.viewportSize.X
	height := t.viewportSize.Y
	t.drawSolidRect(t.viewport, 0, 0, width, 1, borderColor)
	t.drawSolidRect(t.viewport, 0, height-1, width, 1, borderColor)
	t.drawSolidRect(t.viewport, 0, 0, 1, height, borderColor)
	t.drawSolidRect(t.viewport, width-1, 0, 1, height, borderColor)
}

func (t *TextEditor) drawSolidRect(dst *ebiten.Image, x, y, width, height int, fillColor color.Color) {
	if width <= 0 || height <= 0 {
		return
	}
	var options ebiten.DrawImageOptions
	options.GeoM.Scale(float64(width), float64(height))
	options.GeoM.Translate(float64(x), float64(y))
	options.ColorScale.ScaleWithColor(fillColor)
	dst.DrawImage(textEditorSolidPixel, &options)
}

func (t *TextEditor) caretVisible() bool {
	if !t.focused {
		return false
	}
	elapsed := time.Since(t.blinkResetAt)
	return elapsed%(textEditorBlinkPeriod) < (textEditorBlinkPeriod / 2)
}

func (t *TextEditor) caretDrawPosition() (int, int) {
	metrics := t.currentLineMetrics()
	if t.cursorColumn > len(metrics.runes) {
		t.cursorColumn = len(metrics.runes)
	}
	caretX := t.padding + metrics.prefixWidths[t.cursorColumn] - t.scrollX
	caretY := t.padding + (t.cursorRow * t.lineHeight) - t.scrollY
	return caretX, caretY
}

func (t *TextEditor) ensureCaretVisible() {
	innerWidth := maxInt(1, t.widget.Rect.Dx()-(2*t.padding))
	innerHeight := maxInt(1, t.widget.Rect.Dy()-(2*t.padding))
	metrics := t.currentLineMetrics()
	if t.cursorColumn > len(metrics.runes) {
		t.cursorColumn = len(metrics.runes)
	}
	caretX := metrics.prefixWidths[t.cursorColumn]
	caretY := t.cursorRow * t.lineHeight
	nextScrollX := t.scrollX
	nextScrollY := t.scrollY
	if caretX < t.scrollX {
		nextScrollX = caretX
	} else if caretX+textEditorCaretWidth > t.scrollX+innerWidth {
		nextScrollX = caretX + textEditorCaretWidth - innerWidth
	}
	if caretY < t.scrollY {
		nextScrollY = caretY
	} else if caretY+t.lineHeight > t.scrollY+innerHeight {
		nextScrollY = caretY + t.lineHeight - innerHeight
	}
	if t.setScrollOffsets(nextScrollX, nextScrollY) {
		t.renderDirty = true
	}
}

func (t *TextEditor) setCaretFromOffset(offsetX, offsetY int) {
	contentX := maxInt(0, offsetX-t.padding+t.scrollX)
	contentY := maxInt(0, offsetY-t.padding+t.scrollY)
	row := clampInt(contentY/maxInt(1, t.lineHeight), 0, len(t.lines)-1)
	column := t.columnForPixel(row, contentX)
	t.cursorRow = row
	t.cursorColumn = column
	t.desiredColumn = -1
	t.resetBlink()
	t.ensureCaretVisible()
}

func (t *TextEditor) insertText(value string) {
	lineRunes := []rune(t.currentLine())
	insertRunes := []rune(value)
	updated := string(lineRunes[:t.cursorColumn]) + string(insertRunes) + string(lineRunes[t.cursorColumn:])
	t.lines[t.cursorRow] = updated
	t.cursorColumn += len(insertRunes)
	t.desiredColumn = -1
	t.syncDocumentFromLines(true)
}

func (t *TextEditor) insertNewlineWithIndent() {
	lineRunes := []rune(t.currentLine())
	before := string(lineRunes[:t.cursorColumn])
	after := string(lineRunes[t.cursorColumn:])
	indent := leadingSpaces(before)
	nextLine := after
	if strings.TrimSpace(before) == "" && strings.TrimSpace(after) == "" {
		indent = leadingSpaces(t.currentLine())
		nextLine = ""
	} else if strings.HasSuffix(strings.TrimRight(before, " "), ":") {
		indent += t.softTab
	}
	t.lines[t.cursorRow] = before
	insertAt := t.cursorRow + 1
	updatedLines := append([]string{}, t.lines[:insertAt]...)
	updatedLines = append(updatedLines, indent+nextLine)
	updatedLines = append(updatedLines, t.lines[insertAt:]...)
	t.lines = updatedLines
	t.cursorRow = insertAt
	t.cursorColumn = len([]rune(indent))
	t.desiredColumn = -1
	t.syncDocumentFromLines(true)
}

func (t *TextEditor) outdentCurrentLine() {
	line := t.currentLine()
	removed := 0
	if strings.HasPrefix(line, t.softTab) {
		line = strings.TrimPrefix(line, t.softTab)
		removed = len([]rune(t.softTab))
	} else {
		for _, value := range line {
			if value != ' ' || removed >= len([]rune(t.softTab)) {
				break
			}
			removed++
		}
		line = strings.TrimPrefix(line, strings.Repeat(" ", removed))
	}
	if removed == 0 {
		return
	}
	t.lines[t.cursorRow] = line
	t.cursorColumn = maxInt(0, t.cursorColumn-removed)
	t.desiredColumn = -1
	t.syncDocumentFromLines(true)
}

func (t *TextEditor) backspace() {
	if t.cursorColumn > 0 {
		lineRunes := []rune(t.currentLine())
		updated := string(lineRunes[:t.cursorColumn-1]) + string(lineRunes[t.cursorColumn:])
		t.lines[t.cursorRow] = updated
		t.cursorColumn--
		t.desiredColumn = -1
		t.syncDocumentFromLines(true)
		return
	}
	if t.cursorRow == 0 {
		return
	}
	previousRunes := []rune(t.lines[t.cursorRow-1])
	t.lines[t.cursorRow-1] += t.lines[t.cursorRow]
	t.lines = append(t.lines[:t.cursorRow], t.lines[t.cursorRow+1:]...)
	t.cursorRow--
	t.cursorColumn = len(previousRunes)
	t.desiredColumn = -1
	t.syncDocumentFromLines(true)
}

func (t *TextEditor) deleteForward() {
	lineRunes := []rune(t.currentLine())
	if t.cursorColumn < len(lineRunes) {
		updated := string(lineRunes[:t.cursorColumn]) + string(lineRunes[t.cursorColumn+1:])
		t.lines[t.cursorRow] = updated
		t.desiredColumn = -1
		t.syncDocumentFromLines(true)
		return
	}
	if t.cursorRow >= len(t.lines)-1 {
		return
	}
	t.lines[t.cursorRow] += t.lines[t.cursorRow+1]
	t.lines = append(t.lines[:t.cursorRow+1], t.lines[t.cursorRow+2:]...)
	t.desiredColumn = -1
	t.syncDocumentFromLines(true)
}

func (t *TextEditor) moveCursorLeft() {
	if t.cursorColumn > 0 {
		t.cursorColumn--
	} else if t.cursorRow > 0 {
		t.cursorRow--
		t.cursorColumn = len([]rune(t.currentLine()))
	}
	t.desiredColumn = -1
	t.resetBlink()
	if t.focused {
		t.ensureCaretVisible()
	}
}

func (t *TextEditor) moveCursorRight() {
	lineLength := len([]rune(t.currentLine()))
	if t.cursorColumn < lineLength {
		t.cursorColumn++
	} else if t.cursorRow < len(t.lines)-1 {
		t.cursorRow++
		t.cursorColumn = 0
	}
	t.desiredColumn = -1
	t.resetBlink()
	if t.focused {
		t.ensureCaretVisible()
	}
}

func (t *TextEditor) moveCursorVertical(delta int) {
	if t.desiredColumn < 0 {
		t.desiredColumn = t.cursorColumn
	}
	t.cursorRow = clampInt(t.cursorRow+delta, 0, len(t.lines)-1)
	t.cursorColumn = minInt(t.desiredColumn, len([]rune(t.currentLine())))
	t.resetBlink()
	if t.focused {
		t.ensureCaretVisible()
	}
}

func (t *TextEditor) moveCursorHome() {
	t.cursorColumn = 0
	t.desiredColumn = -1
	t.resetBlink()
	if t.focused {
		t.ensureCaretVisible()
	}
}

func (t *TextEditor) moveCursorEnd() {
	t.cursorColumn = len([]rune(t.currentLine()))
	t.desiredColumn = -1
	t.resetBlink()
	if t.focused {
		t.ensureCaretVisible()
	}
}

func (t *TextEditor) scrollByLines(delta int) {
	if delta == 0 {
		return
	}
	if t.setScrollOffsets(t.scrollX, t.scrollY+(delta*t.lineHeight)) {
		t.renderDirty = true
	}
}

func (t *TextEditor) handleMouseWheel(cursor image.Point, wheelY float64) bool {
	if t == nil || t.widget == nil || t.widget.Visibility != widget.Visibility_Show || t.widget.Disabled || wheelY == 0 || !pointInRect(t.widget.Rect, cursor) {
		return false
	}
	lines := -int(math.Round(wheelY * 3))
	if lines == 0 {
		if wheelY > 0 {
			lines = -1
		} else {
			lines = 1
		}
	}
	t.scrollByLines(lines)
	return true
}

func (t *TextEditor) visibleLineCount() int {
	if t == nil || t.lineHeight <= 0 || t.widget == nil {
		return 1
	}
	innerHeight := maxInt(1, t.widget.Rect.Dy()-(2*t.padding))
	count := innerHeight / t.lineHeight
	if count < 1 {
		count = 1
	}
	return count
}

func (t *TextEditor) currentLine() string {
	if len(t.lines) == 0 {
		return ""
	}
	if t.cursorRow < 0 {
		t.cursorRow = 0
	}
	if t.cursorRow >= len(t.lines) {
		t.cursorRow = len(t.lines) - 1
	}
	return t.lines[t.cursorRow]
}

func (t *TextEditor) currentLineMetrics() textEditorLineMetrics {
	if len(t.lineMetrics) == 0 {
		t.rebuildLineMetrics()
	}
	t.cursorRow = clampInt(t.cursorRow, 0, len(t.lineMetrics)-1)
	return t.lineMetrics[t.cursorRow]
}

func (t *TextEditor) clampCursor() {
	if len(t.lines) == 0 {
		t.lines = []string{""}
	}
	t.cursorRow = clampInt(t.cursorRow, 0, len(t.lines)-1)
	t.cursorColumn = clampInt(t.cursorColumn, 0, len(t.currentLineMetrics().runes))
	if t.desiredColumn >= 0 {
		t.desiredColumn = clampInt(t.desiredColumn, 0, len(t.currentLineMetrics().runes))
	}
}

func (t *TextEditor) clampScroll() {
	if t.widget == nil {
		return
	}
	innerWidth := maxInt(1, t.widget.Rect.Dx()-(2*t.padding))
	innerHeight := maxInt(1, t.widget.Rect.Dy()-(2*t.padding))
	maxScrollX := maxInt(0, t.maxLineWidth-innerWidth)
	maxScrollY := maxInt(0, t.contentHeight-innerHeight)
	t.scrollX = clampInt(t.scrollX, 0, maxScrollX)
	t.scrollY = clampInt(t.scrollY, 0, maxScrollY)
}

func (t *TextEditor) setScrollOffsets(scrollX, scrollY int) bool {
	previousX := t.scrollX
	previousY := t.scrollY
	t.scrollX = scrollX
	t.scrollY = scrollY
	t.clampScroll()
	return t.scrollX != previousX || t.scrollY != previousY
}

func (t *TextEditor) columnForPixel(row int, pixelX int) int {
	if row < 0 || row >= len(t.lineMetrics) {
		return 0
	}
	metrics := t.lineMetrics[row]
	if len(metrics.prefixWidths) == 0 || pixelX <= 0 {
		return 0
	}
	if pixelX >= metrics.width {
		return len(metrics.runes)
	}
	left := 0
	right := len(metrics.prefixWidths) - 1
	for left < right {
		middle := left + ((right - left) / 2)
		if metrics.prefixWidths[middle] < pixelX {
			left = middle + 1
		} else {
			right = middle
		}
	}
	column := left
	if column <= 0 {
		return 0
	}
	previousWidth := metrics.prefixWidths[column-1]
	currentWidth := metrics.prefixWidths[column]
	if pixelX-previousWidth <= currentWidth-pixelX {
		return column - 1
	}
	return column
}

func (t *TextEditor) measureTextWidth(text string) int {
	if text == "" {
		return 0
	}
	width, _ := textv2.Measure(text, t.theme.Face, 0)
	return int(math.Ceil(width))
}

func (t *TextEditor) resetBlink() {
	t.blinkResetAt = time.Now()
	t.renderDirty = true
}

func (t *TextEditor) rebuildLineMetrics() {
	if len(t.lines) == 0 {
		t.lines = []string{""}
	}
	t.lineMetrics = make([]textEditorLineMetrics, len(t.lines))
	t.maxLineWidth = 0
	t.blankDocument = true
	for index, line := range t.lines {
		runes := []rune(line)
		prefixWidths := make([]int, len(runes)+1)
		for runeIndex := range runes {
			prefixWidths[runeIndex+1] = t.measureTextWidth(string(runes[:runeIndex+1]))
		}
		width := prefixWidths[len(prefixWidths)-1]
		t.lineMetrics[index] = textEditorLineMetrics{
			text:         line,
			runes:        runes,
			prefixWidths: prefixWidths,
			width:        width,
		}
		if width > t.maxLineWidth {
			t.maxLineWidth = width
		}
		if strings.TrimSpace(line) != "" {
			t.blankDocument = false
		}
	}
	t.contentHeight = len(t.lines) * maxInt(1, t.lineHeight)
}

func leadingSpaces(value string) string {
	runes := []rune(value)
	index := 0
	for index < len(runes) && runes[index] == ' ' {
		index++
	}
	return string(runes[:index])
}

func pointInRect(rect image.Rectangle, point image.Point) bool {
	return point.X >= rect.Min.X && point.X < rect.Max.X && point.Y >= rect.Min.Y && point.Y < rect.Max.Y
}

func textEditorModifierPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyControlLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyControlRight) ||
		ebiten.IsKeyPressed(ebiten.KeyMetaLeft) ||
		ebiten.IsKeyPressed(ebiten.KeyMetaRight)
}

func textEditorShiftPressed() bool {
	return ebiten.IsKeyPressed(ebiten.KeyShiftLeft) || ebiten.IsKeyPressed(ebiten.KeyShiftRight)
}

func textEditorKeyTriggered(key ebiten.Key) bool {
	duration := inpututil.KeyPressDuration(key)
	if duration == 1 {
		return true
	}
	if duration < textEditorRepeatDelay {
		return false
	}
	return (duration-textEditorRepeatDelay)%textEditorRepeatPeriod == 0
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
