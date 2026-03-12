package components

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui/widget"
)

type InspectorState struct {
	Active        bool
	EntityLabel   string
	PrefabPath    string
	DocumentText  string
	Dirty         bool
	ParseError    string
	StatusMessage string
}

type InspectorPanel struct {
	Root            *widget.Container
	TitleText       *widget.Text
	SummaryText     *widget.Text
	StatusText      *widget.Text
	EmptyText       *widget.Text
	Editor          *TextEditor
	onDocumentSaved func(string)
	currentState    InspectorState
}

func NewInspectorPanel(theme *Theme, onDocumentSaved func(string)) *InspectorPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	title := newSectionTitle("Inspector", theme)
	editor := NewTextEditor(theme, nil, onDocumentSaved, widget.WidgetOpts.LayoutData(panelTextLayoutData()))
	panel := &InspectorPanel{
		Root:            root,
		TitleText:       title,
		SummaryText:     newValueText(theme),
		StatusText:      newValueText(theme),
		EmptyText:       newValueText(theme),
		Editor:          editor,
		onDocumentSaved: onDocumentSaved,
	}
	root.AddChild(title)
	root.AddChild(panel.SummaryText)
	root.AddChild(panel.StatusText)
	root.AddChild(panel.EmptyText)
	root.AddChild(panel.Editor)
	return panel
}

func (p *InspectorPanel) Sync(state InspectorState) {
	if p == nil {
		return
	}
	previous := p.currentState
	p.currentState = state
	if p.SummaryText != nil {
		label := state.EntityLabel
		if strings.TrimSpace(label) == "" {
			label = "No entity selected"
		}
		if strings.TrimSpace(state.PrefabPath) != "" {
			label = fmt.Sprintf("%s\nPrefab: %s", label, state.PrefabPath)
		}
		p.SummaryText.Label = label
	}
	if p.StatusText != nil {
		status := strings.TrimSpace(state.StatusMessage)
		if strings.TrimSpace(state.ParseError) != "" {
			status = "Parse error: " + strings.TrimSpace(state.ParseError)
		}
		p.StatusText.Label = status
	}
	if p.EmptyText != nil {
		emptyLabel := strings.TrimSpace(state.StatusMessage)
		if strings.TrimSpace(state.ParseError) != "" {
			emptyLabel = "Parse error: " + strings.TrimSpace(state.ParseError)
		}
		if emptyLabel == "" {
			if state.Active {
				emptyLabel = "No editable components found"
			} else {
				emptyLabel = "Select an entity to inspect"
			}
		}
		p.EmptyText.Label = emptyLabel
	}
	selectionChanged := state.Active != previous.Active || state.EntityLabel != previous.EntityLabel || state.PrefabPath != previous.PrefabPath
	authoritativeUpdate := state.DocumentText != previous.DocumentText || state.Dirty != previous.Dirty
	if p.Editor != nil {
		shouldReload := selectionChanged || !p.Editor.IsFocused() || !p.Editor.IsDirty()
		if shouldReload || (authoritativeUpdate && !state.Dirty) {
			p.Editor.SetText(state.DocumentText)
			p.Editor.SetDirty(state.Dirty)
		}
	}
	hasDocument := strings.TrimSpace(state.DocumentText) != ""
	if p.Editor != nil && !hasDocument {
		p.Editor.Focus(false)
	}
	setWidgetVisible(p.EmptyText, !hasDocument)
	setWidgetVisible(p.Editor, hasDocument)
	setWidgetVisible(p.StatusText, strings.TrimSpace(p.StatusText.Label) != "")
	p.currentState = state
}

func (p *InspectorPanel) AnyInputFocused() bool {
	return p != nil && p.Editor != nil && p.Editor.IsFocused()
}

func (p *InspectorPanel) FocusedInput() *widget.TextInput {
	return nil
}

func (p *InspectorPanel) SetAvailableHeight(height int) bool {
	if p == nil || p.Editor == nil || p.Root == nil || p.Root.GetWidget() == nil || height <= 0 {
		return false
	}
	reserved := 0
	visibleChildren := []widget.PreferredSizeLocateableWidget{}
	for _, child := range []widget.PreferredSizeLocateableWidget{p.TitleText, p.SummaryText, p.StatusText} {
		if child == nil || child.GetWidget() == nil || child.GetWidget().Visibility == widget.Visibility_Hide {
			continue
		}
		visibleChildren = append(visibleChildren, child)
	}
	for _, child := range visibleChildren {
		reserved += inspectorChildHeight(child)
	}
	if len(visibleChildren) > 1 {
		reserved += 8 * (len(visibleChildren) - 1)
	}
	editorHeight := maxInt(textEditorMinHeight, height-reserved-8)
	changed := p.Editor.SetMinHeight(editorHeight)
	rootHeight := reserved + editorHeight + 8
	if p.Root.GetWidget().MinHeight != rootHeight {
		p.Root.GetWidget().MinHeight = rootHeight
		changed = true
	}
	if changed {
		p.Root.RequestRelayout()
	}
	return changed
}

func inspectorChildHeight(child widget.PreferredSizeLocateableWidget) int {
	if child == nil || child.GetWidget() == nil {
		return 0
	}
	if height := child.GetWidget().Rect.Dy(); height > 0 {
		return height
	}
	switch typed := child.(type) {
	case *widget.Text:
		lines := 1 + strings.Count(typed.Label, "\n")
		if lines < 1 {
			lines = 1
		}
		return 22 * lines
	default:
		_, height := child.PreferredSize()
		return height
	}
}

func setWidgetVisible(node widget.PreferredSizeLocateableWidget, visible bool) {
	if node == nil || node.GetWidget() == nil {
		return
	}
	if visible {
		node.GetWidget().Visibility = widget.Visibility_Show
	} else {
		node.GetWidget().Visibility = widget.Visibility_Hide
	}
}
