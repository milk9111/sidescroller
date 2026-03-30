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

func NewUIRoot(w *ecs.World) (ecs.Entity, error) {
	if w == nil {
		return 0, fmt.Errorf("ui root: world is nil")
	}

	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return 0, fmt.Errorf("ui root: load font: %w", err)
	}

	bodyFace := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 20})

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

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
		widget.GraphicOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: false})),
	)
	itemImage.GetWidget().MinWidth = itemPanelImageMinSize
	itemImage.GetWidget().MinHeight = itemPanelImageMinSize
	itemImage.GetWidget().Visibility = widget.Visibility_Hide

	itemText := widget.NewText(
		widget.TextOpts.Text("", &bodyFace, color.NRGBA{R: 236, G: 240, B: 250, A: 255}),
		widget.TextOpts.MaxWidth(itemPanelMaxWidth-56),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)

	portraitBox.AddChild(portrait)
	textPanel.AddChild(text)
	panel.AddChild(textPanel)
	panel.AddChild(portraitBox)
	overlay.AddChild(panel)
	itemPanel.AddChild(itemImage)
	itemPanel.AddChild(itemText)
	itemOverlay.AddChild(itemPanel)
	root.AddChild(overlay)
	root.AddChild(itemOverlay)

	ent := ecs.CreateEntity(w)
	if err := ecs.Add(w, ent, component.PersistentComponent.Kind(), &component.Persistent{ID: "ui_root", KeepOnLevelChange: true, KeepOnReload: false}); err != nil {
		return 0, fmt.Errorf("ui root: add persistent: %w", err)
	}
	if err := ecs.Add(w, ent, component.UIRootComponent.Kind(), &component.UIRoot{UI: &ebitenui.UI{Container: root}}); err != nil {
		return 0, fmt.Errorf("ui root: add root: %w", err)
	}
	if err := ecs.Add(w, ent, component.DialogueUIComponent.Kind(), &component.DialogueUI{Root: root, Overlay: overlay, Panel: panel, PortraitBox: portraitBox, Portrait: portrait, Text: text}); err != nil {
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
	if err := ecs.Add(w, ent, component.DialogueInputComponent.Kind(), &component.DialogueInput{}); err != nil {
		return 0, fmt.Errorf("ui root: add dialogue input: %w", err)
	}

	return ent, nil
}
