package entity

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/ebitenui/ebitenui"
	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"golang.org/x/image/font/gofont/goregular"
)

const dialoguePanelMaxWidth = 560
const dialoguePanelMinHeight = 144
const dialoguePortraitSize = 192
const dialoguePortraitOverlap = 96
const dialoguePortraitFramePadding = 8
const itemPanelMaxWidth = 420
const itemPanelMinHeight = 280
const itemPanelImageMinSize = 96
const inventoryPanelPadding = 0
const inventoryPanelSpacing = 22
const inventoryBodySpacing = 20
const inventoryTitleHeight = 64
const inventoryGridColumns = 4
const inventoryGridVisibleRows = 4
const inventoryGridCellSpacing = 14
const inventorySectionPadding = 18
const inventoryPaneOuterPadding = 24
const inventoryBodyMinHeight = common.BaseHeight - inventoryTitleHeight - inventoryPanelSpacing
const inventoryGridCellSize = (inventoryBodyMinHeight - inventorySectionPadding*2 - (inventoryGridVisibleRows-1)*inventoryGridCellSpacing) / inventoryGridVisibleRows
const inventoryGridPanelMinWidth = inventorySectionPadding*2 + inventoryGridColumns*inventoryGridCellSize + (inventoryGridColumns-1)*inventoryGridCellSpacing
const inventoryGridPanelMinHeight = inventoryBodyMinHeight
const inventoryDetailImageMinSize = 144

func NewUIRoot(w *ecs.World) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("ui root: world is nil")
	}

	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return 0, fmt.Errorf("ui root: load font: %w", err)
	}

	bodyFace := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 20})
	titleFace := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 30})

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	hudLayer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	hudLayer.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}

	overlayLayer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	overlayLayer.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}

	overlay := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{A: 96})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	overlay.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	overlay.GetWidget().Visibility = widget.Visibility_Hide

	panel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	panel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionEnd,
		Padding:            &widget.Insets{Left: 32, Right: 32, Bottom: 28},
	}
	panel.GetWidget().MinWidth = dialoguePanelMaxWidth + dialoguePortraitOverlap
	panel.GetWidget().MinHeight = dialoguePortraitSize + dialoguePortraitFramePadding*2

	textPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{R: 21, G: 23, B: 29, A: 240})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 32 + dialoguePortraitOverlap, Right: 18, Top: 16, Bottom: 16}),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	textPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	textPanel.GetWidget().MinWidth = dialoguePanelMaxWidth
	textPanel.GetWidget().MinHeight = dialoguePanelMinHeight

	portraitBox := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{R: 33, G: 35, B: 44, A: 248})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	portraitBox.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	portraitBox.GetWidget().MinWidth = dialoguePortraitSize + dialoguePortraitFramePadding*2
	portraitBox.GetWidget().MinHeight = dialoguePortraitSize + dialoguePortraitFramePadding*2

	portraitPlaceholder := ebiten.NewImage(1, 1)
	portrait := widget.NewGraphic(
		widget.GraphicOpts.Image(portraitPlaceholder),
		widget.GraphicOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
		})),
	)
	portrait.GetWidget().MinWidth = dialoguePortraitSize
	portrait.GetWidget().MinHeight = dialoguePortraitSize
	portrait.GetWidget().Visibility = widget.Visibility_Hide

	text := widget.NewText(
		widget.TextOpts.Text("", &bodyFace, color.NRGBA{R: 236, G: 240, B: 250, A: 255}),
		widget.TextOpts.MaxWidth(dialoguePanelMaxWidth-dialoguePortraitOverlap-36),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)

	itemOverlay := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{A: 144})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	itemOverlay.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	itemOverlay.GetWidget().Visibility = widget.Visibility_Hide

	itemPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{R: 17, G: 20, B: 25, A: 244})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 28, Right: 28, Top: 24, Bottom: 24}),
			widget.RowLayoutOpts.Spacing(18),
		)),
	)
	itemPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	itemPanel.GetWidget().MinWidth = itemPanelMaxWidth
	itemPanel.GetWidget().MinHeight = itemPanelMinHeight

	itemImage := widget.NewGraphic(
		widget.GraphicOpts.Image(ebiten.NewImage(1, 1)),
		widget.GraphicOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)
	itemImage.GetWidget().MinWidth = itemPanelImageMinSize
	itemImage.GetWidget().MinHeight = itemPanelImageMinSize
	itemImage.GetWidget().Visibility = widget.Visibility_Hide

	itemText := widget.NewText(
		widget.TextOpts.Text("", &bodyFace, color.NRGBA{R: 236, G: 240, B: 250, A: 255}),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionStart),
		widget.TextOpts.MaxWidth(itemPanelMaxWidth-56),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter, Stretch: true})),
	)

	inventoryOverlay := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{R: 5, G: 8, B: 12, A: 196})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	inventoryOverlay.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	inventoryOverlay.GetWidget().Visibility = widget.Visibility_Hide

	inventoryPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{R: 17, G: 23, B: 30, A: 244})),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	inventoryPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		StretchHorizontal: true,
		StretchVertical:   true,
	}

	inventoryTitle := widget.NewText(
		widget.TextOpts.Text("Inventory", &titleFace, color.NRGBA{R: 240, G: 232, B: 214, A: 255}),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter, VerticalPosition: widget.AnchorLayoutPositionStart})),
	)
	inventoryTitle.GetWidget().MinHeight = inventoryTitleHeight

	inventoryBody := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	inventoryBody.GetWidget().LayoutData = widget.AnchorLayoutData{
		StretchHorizontal: true,
		StretchVertical:   true,
		Padding:           &widget.Insets{Top: inventoryTitleHeight + inventoryPanelSpacing, Left: inventoryPanelPadding, Right: inventoryPanelPadding, Bottom: inventoryPanelPadding},
	}
	inventoryBody.GetWidget().MinHeight = inventoryBodyMinHeight

	inventoryGridPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{R: 23, G: 31, B: 40, A: 248})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: inventorySectionPadding, Right: inventorySectionPadding, Top: inventorySectionPadding, Bottom: inventorySectionPadding}),
		)),
	)
	inventoryGridPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		StretchVertical:    true,
		Padding:            &widget.Insets{Left: inventoryPaneOuterPadding, Bottom: inventoryPaneOuterPadding},
	}
	inventoryGridPanel.GetWidget().MinWidth = inventoryGridPanelMinWidth
	inventoryGridPanel.GetWidget().MinHeight = inventoryGridPanelMinHeight

	inventoryGridHost := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	inventoryGridHost.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	inventoryGridHost.GetWidget().MinWidth = inventoryGridPanelMinWidth - inventorySectionPadding*2
	inventoryGridHost.GetWidget().MinHeight = inventoryGridPanelMinHeight - inventorySectionPadding*2

	inventoryDetailPanel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(color.NRGBA{R: 29, G: 38, B: 48, A: 248})),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 24, Right: 24, Top: 24, Bottom: 24}),
			widget.RowLayoutOpts.Spacing(18),
		)),
	)
	inventoryDetailPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		StretchHorizontal: true,
		StretchVertical:   true,
		Padding:           &widget.Insets{Left: inventoryPaneOuterPadding + inventoryGridPanelMinWidth + inventoryBodySpacing, Right: inventoryPaneOuterPadding, Bottom: inventoryPaneOuterPadding},
	}
	inventoryDetailPanel.GetWidget().MinHeight = inventoryBodyMinHeight

	inventoryDetailImage := widget.NewGraphic(
		widget.GraphicOpts.Image(ebiten.NewImage(1, 1)),
		widget.GraphicOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)
	inventoryDetailImage.GetWidget().MinWidth = inventoryDetailImageMinSize
	inventoryDetailImage.GetWidget().MinHeight = inventoryDetailImageMinSize
	inventoryDetailImage.GetWidget().Visibility = widget.Visibility_Hide

	inventoryDetailText := widget.NewText(
		widget.TextOpts.Text("", &bodyFace, color.NRGBA{R: 231, G: 236, B: 242, A: 255}),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionStart),
		widget.TextOpts.MaxWidth(360),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter, Stretch: true})),
	)

	portraitBox.AddChild(portrait)
	textPanel.AddChild(text)
	panel.AddChild(textPanel)
	panel.AddChild(portraitBox)
	overlay.AddChild(panel)
	itemPanel.AddChild(itemImage)
	itemPanel.AddChild(itemText)
	itemOverlay.AddChild(itemPanel)
	inventoryGridPanel.AddChild(inventoryGridHost)
	inventoryDetailPanel.AddChild(inventoryDetailImage)
	inventoryDetailPanel.AddChild(inventoryDetailText)
	inventoryBody.AddChild(inventoryGridPanel)
	inventoryBody.AddChild(inventoryDetailPanel)
	inventoryPanel.AddChild(inventoryTitle)
	inventoryPanel.AddChild(inventoryBody)
	inventoryOverlay.AddChild(inventoryPanel)
	overlayLayer.AddChild(overlay)
	overlayLayer.AddChild(itemOverlay)
	overlayLayer.AddChild(inventoryOverlay)
	root.AddChild(hudLayer)
	root.AddChild(overlayLayer)

	ent := ecs.CreateEntity(w)
	if err := ecs.Add(w, ent, component.PersistentComponent.Kind(), &component.Persistent{ID: "ui_root", KeepOnLevelChange: true, KeepOnReload: false}); err != nil {
		return 0, fmt.Errorf("ui root: add persistent: %w", err)
	}
	if err := ecs.Add(w, ent, component.UIRootComponent.Kind(), &component.UIRoot{UI: &ebitenui.UI{Container: root}}); err != nil {
		return 0, fmt.Errorf("ui root: add root: %w", err)
	}
	if err := ecs.Add(w, ent, component.DialogueUIComponent.Kind(), &component.DialogueUI{Root: root, HUDLayer: hudLayer, OverlayLayer: overlayLayer, Overlay: overlay, Panel: panel, PortraitBox: portraitBox, Portrait: portrait, Text: text}); err != nil {
		return 0, fmt.Errorf("ui root: add dialogue ui: %w", err)
	}
	if err := ecs.Add(w, ent, component.DialogueStateComponent.Kind(), &component.DialogueState{}); err != nil {
		return 0, fmt.Errorf("ui root: add dialogue state: %w", err)
	}
	if err := ecs.Add(w, ent, component.ItemUIComponent.Kind(), &component.ItemUI{Root: root, Overlay: itemOverlay, Panel: itemPanel, Image: itemImage, Text: itemText}); err != nil {
		return 0, fmt.Errorf("ui root: add item ui: %w", err)
	}
	if err := ecs.Add(w, ent, component.ItemStateComponent.Kind(), &component.ItemState{}); err != nil {
		return 0, fmt.Errorf("ui root: add item state: %w", err)
	}
	if err := ecs.Add(w, ent, component.InventoryUIComponent.Kind(), &component.InventoryUI{Root: root, Overlay: inventoryOverlay, Panel: inventoryPanel, Title: inventoryTitle, GridHost: inventoryGridHost, DetailPanel: inventoryDetailPanel, DetailImage: inventoryDetailImage, DetailText: inventoryDetailText}); err != nil {
		return 0, fmt.Errorf("ui root: add inventory ui: %w", err)
	}
	if err := ecs.Add(w, ent, component.InventoryStateComponent.Kind(), &component.InventoryState{}); err != nil {
		return 0, fmt.Errorf("ui root: add inventory state: %w", err)
	}
	if err := ecs.Add(w, ent, component.DialogueInputComponent.Kind(), &component.DialogueInput{}); err != nil {
		return 0, fmt.Errorf("ui root: add dialogue input: %w", err)
	}

	return ent, nil
}
