package components

import (
	"fmt"
	"image"
	"image/color"

	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
)

const (
	maxTilesetPreviewWidth = 232
	tilesetButtonSpacing   = 2
	minTilesetButtonSize   = 16
)

type TilesetPicker struct {
	Root             *widget.Container
	SummaryText      *widget.Text
	gridHost         *widget.Container
	theme            *Theme
	onSelected       func(model.TileSelection)
	imageCache       map[string]*ebiten.Image
	buttons          []*widget.Button
	group            *widget.RadioGroup
	selectedAssetKey string
	selectedPath     string
	selectedIndex    int
	enabled          bool
}

func NewTilesetPicker(theme *Theme, onSelected func(model.TileSelection)) *TilesetPicker {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
		)),
	)
	picker := &TilesetPicker{
		Root:        root,
		SummaryText: newValueText(theme),
		gridHost: widget.NewContainer(
			widget.ContainerOpts.Layout(widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			)),
		),
		theme:         theme,
		onSelected:    onSelected,
		imageCache:    make(map[string]*ebiten.Image),
		selectedIndex: -1,
	}
	root.AddChild(newSectionTitle("Tileset", theme))
	root.AddChild(picker.SummaryText)
	root.AddChild(picker.gridHost)
	return picker
}

func (p *TilesetPicker) Sync(asset *editorio.AssetInfo, selection model.TileSelection, enabled bool) {
	selection = selection.Normalize()
	assetKey := ""
	if asset != nil {
		assetKey = asset.DiskPath
	}
	if p.selectedAssetKey != assetKey {
		p.rebuild(asset)
		p.selectedAssetKey = assetKey
	}
	p.selectedPath = selection.Path
	p.selectedIndex = selection.Index
	if p.enabled != enabled {
		p.enabled = enabled
		p.applyEnabledState()
	}
	p.syncSelection()
	p.syncSummary(asset)
}

func (p *TilesetPicker) SetInteractive(enabled bool) {
	if p == nil {
		return
	}
	if p.enabled != enabled {
		p.enabled = enabled
		p.applyEnabledState()
	}
}

func (p *TilesetPicker) rebuild(asset *editorio.AssetInfo) {
	p.gridHost.RemoveChildren()
	p.buttons = nil
	p.group = nil
	if asset == nil {
		return
	}
	img := p.imageFor(asset.DiskPath)
	if img == nil {
		p.gridHost.AddChild(widget.NewText(
			widget.TextOpts.Text("Preview unavailable", &p.theme.Face, color.NRGBA{R: 176, G: 184, B: 201, A: 255}),
		))
		return
	}
	grid := p.buildGrid(img)
	p.gridHost.AddChild(grid)
	if len(p.buttons) == 0 {
		return
	}
	elements := make([]widget.RadioGroupElement, 0, len(p.buttons))
	for _, button := range p.buttons {
		elements = append(elements, button)
	}
	p.group = widget.NewRadioGroup(
		widget.RadioGroupOpts.Elements(elements...),
		widget.RadioGroupOpts.ChangedHandler(func(args *widget.RadioGroupChangedEventArgs) {
			if args == nil || !p.enabled || asset == nil || p.onSelected == nil {
				return
			}
			for index, button := range p.buttons {
				if args.Active == button {
					p.onSelected(model.TileSelection{Path: asset.Name, Index: index, TileW: model.DefaultTileSize, TileH: model.DefaultTileSize})
					return
				}
			}
		}),
	)
	p.applyEnabledState()
	p.syncSelection()
}

func (p *TilesetPicker) buildGrid(img *ebiten.Image) *widget.Container {
	cols := img.Bounds().Dx() / model.DefaultTileSize
	rows := img.Bounds().Dy() / model.DefaultTileSize
	if cols <= 0 {
		cols = 1
	}
	if rows <= 0 {
		rows = 1
	}
	buttonSize := tileButtonSize(cols)
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(cols),
			widget.GridLayoutOpts.Spacing(tilesetButtonSpacing, tilesetButtonSpacing),
		)),
	)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			src := image.Rect(col*model.DefaultTileSize, row*model.DefaultTileSize, (col+1)*model.DefaultTileSize, (row+1)*model.DefaultTileSize)
			if src.Max.X > img.Bounds().Dx() || src.Max.Y > img.Bounds().Dy() {
				continue
			}
			tileIndex := (row * cols) + col
			sub := img.SubImage(src).(*ebiten.Image)
			button := widget.NewButton(
				widget.ButtonOpts.Image(&widget.ButtonImage{
					Idle:         euiimage.NewAdvancedNineSliceImage(sub, euiimage.NewBorder(0, 0, 0, 0, color.NRGBA{})),
					Hover:        euiimage.NewBorderedNineSliceImage(sub, color.NRGBA{R: 120, G: 180, B: 255, A: 255}, 1),
					Pressed:      euiimage.NewBorderedNineSliceImage(sub, color.NRGBA{R: 255, G: 215, B: 0, A: 255}, 2),
					PressedHover: euiimage.NewBorderedNineSliceImage(sub, color.NRGBA{R: 255, G: 215, B: 0, A: 255}, 2),
					Disabled:     euiimage.NewBorderedNineSliceImage(sub, color.NRGBA{R: 72, G: 78, B: 94, A: 255}, 1),
				}),
				widget.ButtonOpts.ToggleMode(),
				widget.ButtonOpts.WidgetOpts(
					widget.WidgetOpts.MinSize(buttonSize, buttonSize),
				),
				widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
					if p.group != nil && tileIndex >= 0 && tileIndex < len(p.buttons) {
						p.group.SetActive(p.buttons[tileIndex])
					}
				}),
			)
			p.buttons = append(p.buttons, button)
			grid.AddChild(button)
		}
	}
	return grid
}

func (p *TilesetPicker) syncSummary(asset *editorio.AssetInfo) {
	if p.SummaryText == nil {
		return
	}
	if asset == nil {
		p.SummaryText.Label = "No PNG selected"
		return
	}
	status := "Tile picking enabled"
	if !p.enabled {
		status = "Autotile locks tile selection"
	}
	p.SummaryText.Label = fmt.Sprintf("%s · tile %d · %s", asset.Name, max(0, p.selectedIndex), status)
}

func (p *TilesetPicker) syncSelection() {
	if p.group == nil || len(p.buttons) == 0 {
		return
	}
	if p.selectedIndex >= 0 && p.selectedIndex < len(p.buttons) {
		if p.group.Active() != p.buttons[p.selectedIndex] {
			p.group.SetActive(p.buttons[p.selectedIndex])
		}
		return
	}
	if p.group.Active() != nil {
		p.group.SetActive(nil)
	}
}

func (p *TilesetPicker) applyEnabledState() {
	for _, button := range p.buttons {
		if button != nil {
			button.GetWidget().Disabled = !p.enabled
		}
	}
}

func (p *TilesetPicker) imageFor(path string) *ebiten.Image {
	if path == "" {
		return nil
	}
	if img, ok := p.imageCache[path]; ok {
		return img
	}
	img, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		p.imageCache[path] = nil
		return nil
	}
	p.imageCache[path] = img
	return img
}

func tileButtonSize(columns int) int {
	if columns <= 0 {
		return model.DefaultTileSize
	}
	available := maxTilesetPreviewWidth - ((columns - 1) * tilesetButtonSpacing)
	if available <= 0 {
		return minTilesetButtonSize
	}
	size := available / columns
	if size > model.DefaultTileSize {
		size = model.DefaultTileSize
	}
	if size < minTilesetButtonSize {
		size = minTilesetButtonSize
	}
	return size
}
