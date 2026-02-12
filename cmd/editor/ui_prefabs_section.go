package main

import (
	"image/color"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func addPrefabsSection(parent *widget.Container, fontFace *text.Face, prefabs []PrefabInfo, onPrefabSelected func(prefab PrefabInfo)) {
	prefabLabel := widget.NewLabel(
		widget.LabelOpts.Text("Prefabs", fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	parent.AddChild(prefabLabel)

	prefabEntries := make([]any, 0, len(prefabs))
	for _, p := range prefabs {
		prefabEntries = append(prefabEntries, p)
	}
	prefabList := widget.NewList(
		widget.ListOpts.Entries(prefabEntries),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if prefab, ok := e.(PrefabInfo); ok {
				return prefab.Name
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			if onPrefabSelected == nil {
				return
			}
			if prefab, ok := args.Entry.(PrefabInfo); ok {
				onPrefabSelected(prefab)
			}
		}),
	)
	prefabList.GetWidget().MinHeight = 120
	parent.AddChild(prefabList)
}
