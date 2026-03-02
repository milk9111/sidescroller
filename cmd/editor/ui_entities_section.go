package main

import (
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func addEntitiesSection(parent *widget.Container, fontFace *text.Face, entityPanel *EntityPanel, onEntitySelected func(entityIndex int)) {
	entitiesLabel := widget.NewLabel(
		widget.LabelOpts.Text("Active Entities", fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	parent.AddChild(entitiesLabel)

	entityList := widget.NewList(
		widget.ListOpts.Entries([]any{}),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if entry, ok := e.(EntityListEntry); ok {
				return fmt.Sprintf("%d. %s (%d,%d)", entry.Index+1, entry.Type, entry.CellX, entry.CellY)
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			if onEntitySelected == nil || entityPanel == nil || entityPanel.suppressEvents {
				return
			}
			entry, ok := args.Entry.(EntityListEntry)
			if !ok {
				return
			}
			onEntitySelected(entry.Index)
		}),
	)
	configureScrollableList(entityList, 150)
	parent.AddChild(entityList)
	if entityPanel != nil {
		entityPanel.list = entityList
	}
}
