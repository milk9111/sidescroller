package components

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui/widget"
)

type InspectorFieldEdit struct {
	Component string
	Field     string
	Value     string
}

type InspectorFieldState struct {
	Component string
	Field     string
	Label     string
	TypeLabel string
	Value     string
}

type InspectorSectionState struct {
	Component string
	Label     string
	Fields    []InspectorFieldState
}

type InspectorState struct {
	Active      bool
	EntityLabel string
	PrefabPath  string
	Sections    []InspectorSectionState
}

type InspectorPanel struct {
	Root           *widget.Container
	SummaryText    *widget.Text
	EmptyText      *widget.Text
	sectionsRoot   *widget.Container
	inputs         map[string]*widget.TextInput
	structureKey   string
	syncing        bool
	onFieldEdited  func(InspectorFieldEdit)
	currentState   InspectorState
	currentKeyList []string
	theme          *Theme
}

func NewInspectorPanel(theme *Theme, onFieldEdited func(InspectorFieldEdit)) *InspectorPanel {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	panel := &InspectorPanel{
		Root:          root,
		SummaryText:   newValueText(theme),
		EmptyText:     newValueText(theme),
		sectionsRoot:  widget.NewContainer(widget.ContainerOpts.Layout(widget.NewRowLayout(widget.RowLayoutOpts.Direction(widget.DirectionVertical), widget.RowLayoutOpts.Spacing(8)))),
		inputs:        make(map[string]*widget.TextInput),
		onFieldEdited: onFieldEdited,
		theme:         theme,
	}
	root.AddChild(newSectionTitle("Inspector", theme))
	root.AddChild(panel.SummaryText)
	root.AddChild(panel.EmptyText)
	root.AddChild(panel.sectionsRoot)
	return panel
}

func (p *InspectorPanel) Sync(state InspectorState) {
	if p == nil {
		return
	}
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
	structureKey := inspectorStructureKey(state)
	if structureKey != p.structureKey {
		p.rebuild(state)
		p.structureKey = structureKey
	}
	hasSections := len(state.Sections) > 0
	setWidgetVisible(p.EmptyText, !hasSections)
	setWidgetVisible(p.sectionsRoot, hasSections)
	if p.EmptyText != nil {
		if state.Active {
			p.EmptyText.Label = "No editable prefab components found"
		} else {
			p.EmptyText.Label = "Select an entity to inspect"
		}
	}
	p.syncing = true
	defer func() { p.syncing = false }()
	for _, section := range state.Sections {
		for _, field := range section.Fields {
			key := inspectorFieldKey(field.Component, field.Field)
			input := p.inputs[key]
			if input == nil || input.IsFocused() {
				continue
			}
			if input.GetText() != field.Value {
				input.SetText(field.Value)
			}
		}
	}
}

func (p *InspectorPanel) AnyInputFocused() bool {
	return p.FocusedInput() != nil
}

func (p *InspectorPanel) FocusedInput() *widget.TextInput {
	if p == nil {
		return nil
	}
	for _, key := range p.currentKeyList {
		input := p.inputs[key]
		if input != nil && input.IsFocused() {
			return input
		}
	}
	return nil
}

func (p *InspectorPanel) rebuild(state InspectorState) {
	if p == nil || p.sectionsRoot == nil {
		return
	}
	p.sectionsRoot.RemoveChildren()
	p.inputs = make(map[string]*widget.TextInput)
	p.currentKeyList = p.currentKeyList[:0]
	for sectionIndex, section := range state.Sections {
		if sectionIndex > 0 {
			p.sectionsRoot.AddChild(newSeparatorText(p.theme))
		}
		p.sectionsRoot.AddChild(newSectionTitle(section.Label, p.theme))
		for _, field := range section.Fields {
			fieldCopy := field
			label := fieldCopy.Label
			if strings.TrimSpace(fieldCopy.TypeLabel) != "" {
				label = fmt.Sprintf("%s · %s", fieldCopy.Label, fieldCopy.TypeLabel)
			}
			p.sectionsRoot.AddChild(widget.NewText(
				widget.TextOpts.Text(label, &p.theme.Face, p.theme.MutedTextColor),
				widget.TextOpts.MaxWidth(scrollableListMaxWidth),
				widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(panelTextLayoutData())),
			))
			input := newEditorTextInput(p.theme, func(value string) {
				if p.syncing || p.onFieldEdited == nil {
					return
				}
				p.onFieldEdited(InspectorFieldEdit{Component: fieldCopy.Component, Field: fieldCopy.Field, Value: value})
			})
			input.SetText(fieldCopy.Value)
			key := inspectorFieldKey(fieldCopy.Component, fieldCopy.Field)
			p.inputs[key] = input
			p.currentKeyList = append(p.currentKeyList, key)
			p.sectionsRoot.AddChild(input)
		}
	}
	p.Root.RequestRelayout()
	if p.sectionsRoot != nil {
		p.sectionsRoot.RequestRelayout()
	}
}

func inspectorStructureKey(state InspectorState) string {
	parts := make([]string, 0, len(state.Sections)*4)
	for _, section := range state.Sections {
		parts = append(parts, section.Component, section.Label)
		for _, field := range section.Fields {
			parts = append(parts, field.Component, field.Field, field.Label, field.TypeLabel)
		}
	}
	return strings.Join(parts, "|")
}

func inspectorFieldKey(componentName, fieldName string) string {
	return componentName + "." + fieldName
}

func newSeparatorText(theme *Theme) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(strings.Repeat("─", 32), &theme.Face, theme.MutedTextColor),
		widget.TextOpts.MaxWidth(scrollableListMaxWidth),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(panelTextLayoutData())),
	)
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
