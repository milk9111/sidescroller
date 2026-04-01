package system

import (
	"bytes"
	"image/color"
	"strings"
	"sync"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"golang.org/x/image/font/gofont/goregular"
)

const inventoryNavigationThreshold = 0.55
const inventoryTitleHeight = 64
const inventoryGridColumns = 4
const inventoryGridVisibleRows = 4
const inventoryGridVisibleSlots = inventoryGridColumns * inventoryGridVisibleRows
const inventoryCellSpacing = 14
const inventorySectionPadding = 18
const inventoryPanelSpacing = 22
const inventoryBodyMinHeight = common.BaseHeight - inventoryTitleHeight - inventoryPanelSpacing
const inventoryCellSize = (inventoryBodyMinHeight - inventorySectionPadding*2 - (inventoryGridVisibleRows-1)*inventoryCellSpacing) / inventoryGridVisibleRows
const inventoryCellIconSize = inventoryCellSize - 32

type InventorySystem struct{}

type inventoryViewItem struct {
	Prefab      string
	Name        string
	Description string
	Count       int
	Image       *ebiten.Image
}

var (
	inventoryTextFaceOnce sync.Once
	inventoryTextFace     textv2.Face
	inventoryTextFaceErr  error
)

func NewInventorySystem() *InventorySystem {
	return &InventorySystem{}
}

func IsInventoryActive(w *ecs.World) bool {
	_, state, _, _, ok := inventoryUIState(w)
	return ok && state != nil && state.Active
}

func (s *InventorySystem) Update(w *ecs.World) {
	stateEnt, state, ui, inventory, ok := inventoryUIState(w)
	if !ok || state == nil || ui == nil || inventory == nil {
		return
	}

	if !state.Active {
		hideInventoryUI(ui)
		return
	}

	items := inventoryViewItems(inventory)
	if len(items) == 0 {
		state.SelectedIndex = 0
	} else if state.SelectedIndex < 0 {
		state.SelectedIndex = 0
	} else if state.SelectedIndex >= len(items) {
		state.SelectedIndex = len(items) - 1
	}

	if input := activePlayerInput(w); input != nil {
		if input.MenuPressed {
			state.Active = false
			state.LastMoveX = 0
			state.LastMoveY = 0
			hideInventoryUI(ui)
			_ = ecs.Add(w, stateEnt, component.InventoryStateComponent.Kind(), state)
			return
		}

		navigateInventory(state, input, len(items))
	}

	showInventory(ui, items, state.SelectedIndex)
	_ = ecs.Add(w, stateEnt, component.InventoryStateComponent.Kind(), state)
}

func inventoryUIState(w *ecs.World) (ecs.Entity, *component.InventoryState, *component.InventoryUI, *component.Inventory, bool) {
	if w == nil {
		return 0, nil, nil, nil, false
	}

	uiEnt, ok := ecs.First(w, component.InventoryStateComponent.Kind())
	if !ok {
		return 0, nil, nil, nil, false
	}

	state, ok := ecs.Get(w, uiEnt, component.InventoryStateComponent.Kind())
	if !ok || state == nil {
		return 0, nil, nil, nil, false
	}

	ui, ok := ecs.Get(w, uiEnt, component.InventoryUIComponent.Kind())
	if !ok || ui == nil {
		return 0, nil, nil, nil, false
	}

	inventory := currentPlayerInventory(w)
	if inventory == nil {
		return 0, nil, nil, nil, false
	}

	return uiEnt, state, ui, inventory, true
}

func activePlayerInput(w *ecs.World) *component.Input {
	if w == nil {
		return nil
	}

	ent, ok := ecs.First(w, component.InputComponent.Kind())
	if !ok {
		return nil
	}

	input, ok := ecs.Get(w, ent, component.InputComponent.Kind())
	if !ok || input == nil {
		return nil
	}

	return input
}

func inventoryViewItems(inventory *component.Inventory) []inventoryViewItem {
	if inventory == nil || len(inventory.Items) == 0 {
		return nil
	}

	items := make([]inventoryViewItem, 0, len(inventory.Items))
	for _, entry := range inventory.Items {
		if entry.Count <= 0 || strings.TrimSpace(entry.Prefab) == "" {
			continue
		}
		definition, err := resolveInventoryItemDefinition(entry.Prefab)
		if err != nil || definition == nil {
			continue
		}
		items = append(items, inventoryViewItem{
			Prefab:      definition.Prefab,
			Name:        strings.TrimSpace(definition.Name),
			Description: strings.TrimSpace(definition.Description),
			Count:       entry.Count,
			Image:       definition.Image,
		})
	}
	return items
}

func navigateInventory(state *component.InventoryState, input *component.Input, itemCount int) {
	if state == nil || input == nil || itemCount <= 0 {
		if state != nil {
			state.LastMoveX = 0
			state.LastMoveY = 0
		}
		return
	}

	moveX := inventoryAxisDirection(input.MoveX)
	moveY := inventoryAxisDirection(input.MoveY)

	if moveX != 0 && moveX != state.LastMoveX {
		state.SelectedIndex = moveInventorySelection(state.SelectedIndex, moveX, 0, itemCount)
	}
	if moveY != 0 && moveY != state.LastMoveY {
		state.SelectedIndex = moveInventorySelection(state.SelectedIndex, 0, moveY, itemCount)
	}

	state.LastMoveX = moveX
	state.LastMoveY = moveY
}

func inventoryAxisDirection(v float64) int {
	if v <= -inventoryNavigationThreshold {
		return -1
	}
	if v >= inventoryNavigationThreshold {
		return 1
	}
	return 0
}

func moveInventorySelection(index, dx, dy, itemCount int) int {
	if itemCount <= 0 {
		return 0
	}
	if index < 0 {
		index = 0
	}
	if index >= itemCount {
		index = itemCount - 1
	}

	if dx != 0 {
		next := index + dx
		if next >= 0 && next < itemCount && next/inventoryGridColumns == index/inventoryGridColumns {
			index = next
		}
	}

	if dy != 0 {
		col := index % inventoryGridColumns
		target := index + (dy * inventoryGridColumns)
		if dy < 0 {
			if target >= 0 {
				index = target
			}
		} else {
			if target < itemCount {
				index = target
			} else {
				lastRowStart := ((itemCount - 1) / inventoryGridColumns) * inventoryGridColumns
				candidate := lastRowStart + col
				if candidate >= itemCount {
					candidate = itemCount - 1
				}
				index = candidate
			}
		}
	}

	return index
}

func hideInventoryUI(ui *component.InventoryUI) {
	if ui == nil {
		return
	}
	if ui.DetailText != nil {
		ui.DetailText.Label = ""
	}
	if ui.DetailImage != nil {
		ui.DetailImage.Image = ebiten.NewImage(1, 1)
		setWidgetVisible(ui.DetailImage, false)
	}
	if ui.GridHost != nil {
		ui.GridHost.RemoveChildren()
	}
	setWidgetVisible(ui.Overlay, false)
	requestInventoryUIRelayout(ui)
}

func showInventory(ui *component.InventoryUI, items []inventoryViewItem, selectedIndex int) {
	if ui == nil {
		return
	}
	if ui.GridHost != nil {
		ui.GridHost.RemoveChildren()
		if len(items) == 0 {
			empty := widget.NewText(
				widget.TextOpts.Text("No items collected.", inventoryTextFaceRef(), color.NRGBA{R: 181, G: 190, B: 198, A: 255}),
				widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
				widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{HorizontalPosition: widget.AnchorLayoutPositionCenter, VerticalPosition: widget.AnchorLayoutPositionCenter})),
			)
			ui.GridHost.AddChild(empty)
		} else {
			totalSlots := max(len(items), inventoryGridVisibleSlots)
			grid := widget.NewContainer(
				widget.ContainerOpts.Layout(widget.NewGridLayout(
					widget.GridLayoutOpts.Columns(inventoryGridColumns),
					widget.GridLayoutOpts.Spacing(inventoryCellSpacing, inventoryCellSpacing),
				)),
			)
			for index := 0; index < totalSlots; index++ {
				if index < len(items) {
					item := items[index]
					grid.AddChild(buildInventoryCell(&item, index == selectedIndex))
					continue
				}
				grid.AddChild(buildInventoryCell(nil, false))
			}
			ui.GridHost.AddChild(grid)
		}
	}

	if len(items) > 0 && selectedIndex >= 0 && selectedIndex < len(items) {
		selected := items[selectedIndex]
		if ui.DetailImage != nil {
			if selected.Image != nil {
				ui.DetailImage.Image = scaleInventoryImage(selected.Image, 1.5)
				setWidgetVisible(ui.DetailImage, true)
			} else {
				setWidgetVisible(ui.DetailImage, false)
			}
		}
		if ui.DetailText != nil {
			ui.DetailText.Label = selected.Description
		}
	} else {
		if ui.DetailImage != nil {
			setWidgetVisible(ui.DetailImage, false)
		}
		if ui.DetailText != nil {
			ui.DetailText.Label = ""
		}
	}

	setWidgetVisible(ui.Overlay, true)
	requestInventoryUIRelayout(ui)
}

func buildInventoryCell(item *inventoryViewItem, selected bool) *widget.Container {
	background := color.NRGBA{R: 28, G: 35, B: 45, A: 255}
	underlineColor := color.NRGBA{R: 236, G: 191, B: 99, A: 255}
	countColor := color.NRGBA{R: 240, G: 232, B: 214, A: 255}
	if selected {
		background = color.NRGBA{R: 47, G: 58, B: 72, A: 255}
	}

	cell := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(background)),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Padding(&widget.Insets{Left: 8, Right: 8, Top: 8, Bottom: 8}),
			widget.RowLayoutOpts.Spacing(6),
		)),
	)
	cell.GetWidget().MinWidth = inventoryCellSize
	cell.GetWidget().MinHeight = inventoryCellSize
	if item == nil {
		return cell
	}

	icon := widget.NewGraphic(
		widget.GraphicOpts.Image(imageOrPlaceholder(fitInventoryImage(item.Image, inventoryCellIconSize, inventoryCellIconSize))),
		widget.GraphicOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)
	icon.GetWidget().MinWidth = inventoryCellIconSize
	icon.GetWidget().MinHeight = inventoryCellIconSize
	if item.Image == nil {
		setWidgetVisible(icon, false)
	}

	countLabel := ""
	if item.Count > 1 {
		countLabel = strings.TrimSpace("x" + itoa(item.Count))
	}
	count := widget.NewText(
		widget.TextOpts.Text(countLabel, inventoryTextFaceRef(), countColor),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)

	underline := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(euiimage.NewNineSliceColor(underlineColor)),
	)
	underline.GetWidget().MinHeight = 4
	underline.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	setWidgetVisible(underline, selected)

	cell.AddChild(icon)
	cell.AddChild(count)
	cell.AddChild(underline)
	return cell
}

func imageOrPlaceholder(img *ebiten.Image) *ebiten.Image {
	if img != nil {
		return img
	}
	return ebiten.NewImage(1, 1)
}

func fitInventoryImage(img *ebiten.Image, maxWidth, maxHeight int) *ebiten.Image {
	if img == nil || maxWidth <= 0 || maxHeight <= 0 {
		return img
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return img
	}
	if width <= maxWidth && height <= maxHeight {
		return img
	}
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	if scale <= 0 || scale >= 1 {
		return img
	}
	dst := ebiten.NewImage(int(float64(width)*scale), int(float64(height)*scale))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	dst.DrawImage(img, op)
	return dst
}

func requestInventoryUIRelayout(ui *component.InventoryUI) {
	if ui == nil {
		return
	}
	if ui.Root != nil {
		ui.Root.RequestRelayout()
	}
	if ui.Overlay != nil {
		ui.Overlay.RequestRelayout()
	}
	if ui.Panel != nil {
		ui.Panel.RequestRelayout()
	}
	if ui.GridHost != nil {
		ui.GridHost.RequestRelayout()
	}
	if ui.DetailPanel != nil {
		ui.DetailPanel.RequestRelayout()
	}
}

func inventoryTextFaceRef() *textv2.Face {
	face, _ := buildInventoryTextFace()
	return &face
}

func buildInventoryTextFace() (textv2.Face, error) {
	inventoryTextFaceOnce.Do(func() {
		fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
		if err != nil {
			inventoryTextFaceErr = err
			return
		}
		inventoryTextFace = textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 20})
	})
	return inventoryTextFace, inventoryTextFaceErr
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	negative := v < 0
	if negative {
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		buf = append(buf, byte('0'+(v%10)))
		v /= 10
	}
	if negative {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
