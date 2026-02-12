package main

import (
	"bytes"
	"fmt"
	"image/color"
	"time"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/event"
	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

// BuildEditorUI creates the root UI container with a right panel for the asset list.
// solidNineSlice returns a solid color *image.NineSlice for widget backgrounds.
func solidNineSlice(c color.Color) *image.NineSlice {
	return image.NewNineSliceColor(c)
}

type ToolBar struct {
	group   *widget.RadioGroup
	buttons []*widget.Button
}

type TransitionUI struct {
	modeBtn *widget.Button
	form    *widget.Container

	idInput     *widget.TextInput
	levelInput  *widget.TextInput
	linkedInput *widget.TextInput
	dirCombo    *widget.ComboButton

	suppress bool
	modeOn   bool
}

func (t *TransitionUI) SetMode(enabled bool) {
	if t == nil {
		return
	}
	t.modeOn = enabled
	if t.modeBtn == nil {
		return
	}
	label := "Transitions: Off"
	if enabled {
		label = "Transitions: On"
	}
	if text := t.modeBtn.Text(); text != nil {
		text.Label = label
	}
}

func (t *TransitionUI) SetFormVisible(visible bool) {
	if t == nil || t.form == nil {
		return
	}
	if visible {
		t.form.GetWidget().Visibility = widget.Visibility_Show
	} else {
		t.form.GetWidget().Visibility = widget.Visibility_Hide
	}
}

func (t *TransitionUI) SetFields(id, level, linked, dir string) {
	if t == nil {
		return
	}
	t.suppress = true
	if t.idInput != nil {
		t.idInput.SetText(id)
	}
	if t.levelInput != nil {
		t.levelInput.SetText(level)
	}
	if t.linkedInput != nil {
		t.linkedInput.SetText(linked)
	}
	if t.dirCombo != nil {
		label := dir
		if label == "" {
			label = "(none)"
		}
		t.dirCombo.SetLabel(label)
	}
	t.suppress = false
}

// Save dialog removed.

func (tb *ToolBar) SetTool(t Tool) {
	idx := int(t)
	if tb == nil || tb.group == nil || idx < 0 || idx >= len(tb.buttons) {
		return
	}
	tb.group.SetActive(tb.buttons[idx])
}

func BuildEditorUI(
	assets []AssetInfo,
	prefabs []PrefabInfo,
	onAssetSelected func(asset AssetInfo, setTileset func(img *ebiten.Image)),
	onToolSelected func(tool Tool),
	onTileSelected func(tileIndex int),
	onLayerSelected func(layerIndex int),
	onLayerRenamed func(layerIndex int, newName string),
	onNewLayer func(),
	onMoveLayerUp func(layerIndex int),
	onMoveLayerDown func(layerIndex int),
	onTogglePhysics func(),
	onTogglePhysicsHighlight func(),
	onToggleAutotile func(),
	onPrefabSelected func(prefab PrefabInfo),
	onToggleTransitionMode func(enabled bool),
	onTransitionFieldChanged func(field, value string),
	initialLayers []string,
	initialLayerIndex int,
	initialTool Tool,
	initialAutotileEnabled bool,
) (*ebitenui.UI, *ToolBar, *LayerPanel, *widget.TextInput, func(img *ebiten.Image), func(tileIndex int), func(enabled bool), *TransitionUI) {
	ui := &ebitenui.UI{}

	s, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic("Failed to load font: " + err.Error())
	}

	var fontFace text.Face = &text.GoTextFace{Source: s, Size: 14}

	ui.PrimaryTheme = &widget.Theme{
		ListTheme: &widget.ListParams{
			EntryFace: &fontFace,
			EntryColor: &widget.ListEntryColor{
				Unselected:          color.Black,
				Selected:            color.RGBA{0, 0, 128, 255},
				DisabledUnselected:  color.Gray{Y: 128},
				DisabledSelected:    color.Gray{Y: 64},
				SelectingBackground: color.RGBA{200, 220, 255, 255},
				SelectedBackground:  color.RGBA{180, 200, 255, 255},
			},
			ScrollContainerImage: &widget.ScrollContainerImage{
				Idle: solidNineSlice(color.RGBA{220, 220, 220, 255}),
				Mask: solidNineSlice(color.RGBA{220, 220, 220, 255}),
			},
		},
		PanelTheme: &widget.PanelParams{
			BackgroundImage: solidNineSlice(color.RGBA{40, 40, 40, 255}),
		},
		ButtonTheme: &widget.ButtonParams{
			Image: &widget.ButtonImage{
				Idle:    solidNineSlice(color.RGBA{180, 180, 180, 255}),
				Hover:   solidNineSlice(color.RGBA{200, 200, 200, 255}),
				Pressed: solidNineSlice(color.RGBA{160, 160, 160, 255}),
			},
			TextFace: &fontFace,
			TextColor: &widget.ButtonTextColor{
				Idle: color.Black,
			},
		},
		SliderTheme: &widget.SliderParams{
			TrackImage: &widget.SliderTrackImage{
				Idle:  solidNineSlice(color.RGBA{180, 180, 180, 255}),
				Hover: solidNineSlice(color.RGBA{200, 200, 200, 255}),
			},
			HandleImage: &widget.ButtonImage{
				Idle:    solidNineSlice(color.RGBA{120, 120, 120, 255}),
				Hover:   solidNineSlice(color.RGBA{160, 160, 160, 255}),
				Pressed: solidNineSlice(color.RGBA{100, 100, 100, 255}),
			},
		},
	}

	var tilesetImg *ebiten.Image
	var tileGridZoom *TilesetGridZoomable

	// Asset list entries
	var entries []any
	if len(assets) > 0 {
		entries = make([]any, len(assets))
		for i, a := range assets {
			entries[i] = a
		}
	} else {
		entries = []any{}
	}

	// Tileset panel: vertical layout (top: asset list, bottom: tileset grid)
	tilesetPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(240, 400),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{40, 40, 40, 255})),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)

	// helper to apply a tileset image into the tileset panel
	applyTileset := func(img *ebiten.Image) {
		tilesetImg = img
		if tileGridZoom != nil {
			tilesetPanel.RemoveChild(tileGridZoom.Container)
		}
		tileGridZoom = NewTilesetGridZoomable(tilesetImg, 32, func(tileIndex int) {
			if onTileSelected != nil {
				onTileSelected(tileIndex)
			}
		})
		tilesetPanel.AddChild(tileGridZoom.Container)
	}

	setTilesetSelection := func(tileIndex int) {
		if tileGridZoom == nil {
			return
		}
		tileGridZoom.SetSelected(tileIndex)
	}

	setTilesetSelectionEnabled := func(enabled bool) {
		if tileGridZoom == nil {
			return
		}
		tileGridZoom.SetSelectionEnabled(enabled)
	}

	transitionUI := &TransitionUI{}

	// Asset list (scrollable, top half, fixed height)
	assetList := widget.NewList(
		widget.ListOpts.Entries(entries),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if asset, ok := e.(AssetInfo); ok {
				return asset.Name
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			if asset, ok := args.Entry.(AssetInfo); ok {
				onAssetSelected(asset, applyTileset)
			}
		}),
	)
	// No MinHeight, let layout engine handle sizing
	tilesetPanel.AddChild(assetList)
	// tileGrid will be added after asset selection

	// --- Floating Toolbar ---
	toolNames := []string{"Brush", "Erase", "Fill", "Line"}
	buttonTextColor := &widget.ButtonTextColor{
		Idle:     color.Black,
		Hover:    color.Black,
		Pressed:  color.RGBA{0, 0, 200, 255},
		Disabled: color.Gray{Y: 128},
	}
	toolbar := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(220, 48),
		),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{220, 220, 240, 255})),
	)
	var toolButtons []*widget.Button
	for _, name := range toolNames {
		btn := widget.NewButton(
			widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
			widget.ButtonOpts.Text(name, &fontFace, buttonTextColor),
			widget.ButtonOpts.ToggleMode(),
			widget.ButtonOpts.WidgetOpts(
				widget.WidgetOpts.MinSize(48, 40),
			),
		)
		toolButtons = append(toolButtons, btn)
		toolbar.AddChild(btn)
	}

	// (debug overlay removed)

	elements := make([]widget.RadioGroupElement, 0, len(toolButtons))
	for _, b := range toolButtons {
		elements = append(elements, b)
	}

	group := widget.NewRadioGroup(
		widget.RadioGroupOpts.Elements(elements...),
		widget.RadioGroupOpts.ChangedHandler(func(args *widget.RadioGroupChangedEventArgs) {
			if onToolSelected == nil {
				return
			}
			for idx, b := range toolButtons {
				if args.Active == b {
					onToolSelected(Tool(idx))
					return
				}
			}
		}),
	)

	if idx := int(initialTool); idx >= 0 && idx < len(toolButtons) {
		group.SetActive(toolButtons[idx])
	}

	// --- Left Panel (Layers) ---
	layerPanel := NewLayerPanel()
	layerPanel.onNewLayer = onNewLayer
	layerPanel.onMoveUp = onMoveLayerUp
	layerPanel.onMoveDown = onMoveLayerDown

	leftPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(200, 400),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{40, 40, 40, 255})),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)

	// Filename input at top of left panel
	fileLabel := widget.NewLabel(
		widget.LabelOpts.Text("File", &fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	fileNameInput := widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(180, 28),
		),
		widget.TextInputOpts.Image(&widget.TextInputImage{
			Idle:     solidNineSlice(color.RGBA{245, 245, 245, 255}),
			Disabled: solidNineSlice(color.RGBA{200, 200, 200, 255}),
		}),
		widget.TextInputOpts.Color(&widget.TextInputColor{
			Idle:     color.Black,
			Disabled: color.Gray{Y: 120},
			Caret:    color.Black,
		}),
		widget.TextInputOpts.Face(&fontFace),
	)
	leftPanel.AddChild(fileLabel)
	leftPanel.AddChild(fileNameInput)

	layersLabel := widget.NewLabel(
		widget.LabelOpts.Text("Layers", &fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	leftPanel.AddChild(layersLabel)

	layerList := widget.NewList(
		widget.ListOpts.Entries([]any{}),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if entry, ok := e.(LayerEntry); ok {
				return fmt.Sprintf("%d. %s", entry.Index+1, entry.Name)
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			entry, ok := args.Entry.(LayerEntry)
			if !ok {
				return
			}
			now := time.Now()
			// If we're suppressing programmatic events, record the selection time
			// but don't interpret it as a user click / double-click.
			if layerPanel.suppressEvents {
				layerPanel.lastClickIndex = entry.Index
				layerPanel.lastClickTime = now
				if onLayerSelected != nil {
					onLayerSelected(entry.Index)
				}
				return
			}

			layerPanel.lastClickIndex = entry.Index
			layerPanel.lastClickTime = now
			if onLayerSelected != nil {
				onLayerSelected(entry.Index)
			}
		}),
	)
	leftPanel.AddChild(layerList)
	layerPanel.list = layerList

	buttonsRow := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(6),
			),
		),
	)
	newLayerBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("New", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if layerPanel.onNewLayer != nil {
				layerPanel.onNewLayer()
			}
		}),
	)
	upBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Up", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if layerPanel.onMoveUp == nil {
				return
			}
			if sel, ok := layerList.SelectedEntry().(LayerEntry); ok {
				layerPanel.onMoveUp(sel.Index)
			}
		}),
	)
	downBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Down", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if layerPanel.onMoveDown == nil {
				return
			}
			if sel, ok := layerList.SelectedEntry().(LayerEntry); ok {
				layerPanel.onMoveDown(sel.Index)
			}
		}),
	)
	renameBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Rename", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
	)
	buttonsRow.AddChild(newLayerBtn)
	buttonsRow.AddChild(upBtn)
	buttonsRow.AddChild(downBtn)
	buttonsRow.AddChild(renameBtn)
	leftPanel.AddChild(buttonsRow)

	physicsButtonsRow := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(6),
			),
		),
	)
	physicsBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Physics Off", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if onTogglePhysics != nil {
				onTogglePhysics()
			}
		}),
	)
	autotileBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Autotile On", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if onToggleAutotile != nil {
				onToggleAutotile()
			}
		}),
	)
	highlightBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Highlight Physics", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			if onTogglePhysicsHighlight != nil {
				onTogglePhysicsHighlight()
			}
		}),
	)
	physicsButtonsRow.AddChild(physicsBtn)
	physicsButtonsRow.AddChild(autotileBtn)
	physicsButtonsRow.AddChild(highlightBtn)
	leftPanel.AddChild(physicsButtonsRow)
	layerPanel.physicsBtn = physicsBtn
	layerPanel.autotileBtn = autotileBtn

	prefabLabel := widget.NewLabel(
		widget.LabelOpts.Text("Prefabs", &fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	leftPanel.AddChild(prefabLabel)

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
	leftPanel.AddChild(prefabList)

	// --- Transition placement + properties ---
	transitionLabel := widget.NewLabel(
		widget.LabelOpts.Text("Transitions", &fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	)
	leftPanel.AddChild(transitionLabel)

	transitionModeBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Transitions: Off", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			transitionUI.modeOn = !transitionUI.modeOn
			transitionUI.SetMode(transitionUI.modeOn)
			if onToggleTransitionMode != nil {
				onToggleTransitionMode(transitionUI.modeOn)
			}
		}),
	)
	transitionUI.modeBtn = transitionModeBtn

	transitionForm := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(6),
			),
		),
	)
	transitionForm.GetWidget().Visibility = widget.Visibility_Hide
	transitionUI.form = transitionForm

	// Group the button and its form in a container so the form appears
	// directly below the button in the left panel.
	transitionContainer := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(4),
			),
		),
	)
	transitionContainer.AddChild(transitionModeBtn)
	transitionContainer.AddChild(transitionForm)
	leftPanel.AddChild(transitionContainer)

	makeField := func(label string, onChange func(text string)) *widget.TextInput {
		transitionForm.AddChild(widget.NewLabel(
			widget.LabelOpts.Text(label, &fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
		))
		input := widget.NewTextInput(
			widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.MinSize(180, 28)),
			widget.TextInputOpts.Image(&widget.TextInputImage{
				Idle:     solidNineSlice(color.RGBA{245, 245, 245, 255}),
				Disabled: solidNineSlice(color.RGBA{200, 200, 200, 255}),
			}),
			widget.TextInputOpts.Color(&widget.TextInputColor{Idle: color.Black, Disabled: color.Gray{Y: 120}, Caret: color.Black}),
			widget.TextInputOpts.Face(&fontFace),
			widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
				if transitionUI.suppress {
					return
				}
				if onChange != nil {
					onChange(args.InputText)
				}
			}),
		)
		transitionForm.AddChild(input)
		return input
	}

	transitionUI.idInput = makeField("ID", func(text string) {
		if onTransitionFieldChanged != nil {
			onTransitionFieldChanged("id", text)
		}
	})
	transitionUI.levelInput = makeField("To level", func(text string) {
		if onTransitionFieldChanged != nil {
			onTransitionFieldChanged("to_level", text)
		}
	})
	transitionUI.linkedInput = makeField("Linked transition ID", func(text string) {
		if onTransitionFieldChanged != nil {
			onTransitionFieldChanged("linked_id", text)
		}
	})
	// enter_dir dropdown
	transitionForm.AddChild(widget.NewLabel(
		widget.LabelOpts.Text("Enter direction", &fontFace, &widget.LabelColor{Idle: color.White, Disabled: color.Gray{Y: 140}}),
	))
	dirEntries := []any{"(none)", "up", "down", "left", "right"}
	dirList := widget.NewList(
		widget.ListOpts.Entries(dirEntries),
		widget.ListOpts.EntryLabelFunc(func(e any) string {
			if s, ok := e.(string); ok {
				return s
			}
			return ""
		}),
		widget.ListOpts.EntrySelectedHandler(func(args *widget.ListEntrySelectedEventArgs) {
			if transitionUI.suppress {
				return
			}
			val, _ := args.Entry.(string)
			label := val
			if label == "" {
				label = "(none)"
			}
			if transitionUI.dirCombo != nil {
				transitionUI.dirCombo.SetLabel(label)
				transitionUI.dirCombo.ContentVisible = false
			}
			if onTransitionFieldChanged != nil {
				if val == "(none)" {
					val = ""
				}
				onTransitionFieldChanged("enter_dir", val)
			}
		}),
	)
	dirList.GetWidget().MinHeight = 80

	dirCombo := widget.NewComboButton(
		widget.ComboButtonOpts.ButtonOpts(
			widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
			widget.ButtonOpts.Text("(none)", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		),
		widget.ComboButtonOpts.Content(dirList),
		widget.ComboButtonOpts.MaxContentHeight(120),
	)
	transitionUI.dirCombo = dirCombo
	transitionForm.AddChild(dirCombo)

	// Note: transitionForm is already a child of transitionContainer.
	// Adding it again to leftPanel would make it render/layout incorrectly.

	// Rename dialog (modal overlay)
	var renameIdx int = -1
	renameOverlay := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				StretchHorizontal:  true,
				StretchVertical:    true,
			}),
			widget.WidgetOpts.MinSize(1, 1),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{0, 0, 0, 160})),
	)
	renameOverlay.GetWidget().Visibility = widget.Visibility_Hide

	dialog := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(320, 140),
		),
		widget.ContainerOpts.BackgroundImage(solidNineSlice(color.RGBA{220, 220, 220, 255})),
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionVertical),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)
	dialog.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}

	nameLabel := widget.NewLabel(
		widget.LabelOpts.Text("Rename layer", &fontFace, &widget.LabelColor{Idle: color.Black, Disabled: color.Gray{Y: 140}}),
	)
	nameInput := widget.NewTextInput(
		widget.TextInputOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(260, 28),
		),
		widget.TextInputOpts.Image(&widget.TextInputImage{
			Idle:     solidNineSlice(color.RGBA{245, 245, 245, 255}),
			Disabled: solidNineSlice(color.RGBA{200, 200, 200, 255}),
		}),
		widget.TextInputOpts.Color(&widget.TextInputColor{
			Idle:     color.Black,
			Disabled: color.Gray{Y: 120},
			Caret:    color.Black,
		}),
		widget.TextInputOpts.Face(&fontFace),
		widget.TextInputOpts.SubmitOnEnter(true),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if renameIdx >= 0 && onLayerRenamed != nil && args.InputText != "" {
				onLayerRenamed(renameIdx, args.InputText)
			}
			renameOverlay.GetWidget().Visibility = widget.Visibility_Hide
			renameIdx = -1
		}),
	)

	buttonsRow2 := widget.NewContainer(
		widget.ContainerOpts.Layout(
			widget.NewRowLayout(
				widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
				widget.RowLayoutOpts.Spacing(8),
			),
		),
	)
	okBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("OK", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			text := nameInput.GetText()
			if renameIdx >= 0 && onLayerRenamed != nil && text != "" {
				onLayerRenamed(renameIdx, text)
			}
			renameOverlay.GetWidget().Visibility = widget.Visibility_Hide
			renameIdx = -1
		}),
	)
	cancelBtn := widget.NewButton(
		widget.ButtonOpts.Image(ui.PrimaryTheme.ButtonTheme.Image),
		widget.ButtonOpts.Text("Cancel", &fontFace, ui.PrimaryTheme.ButtonTheme.TextColor),
		widget.ButtonOpts.ClickedHandler(func(args *widget.ButtonClickedEventArgs) {
			renameOverlay.GetWidget().Visibility = widget.Visibility_Hide
			renameIdx = -1
		}),
	)
	buttonsRow2.AddChild(okBtn)
	buttonsRow2.AddChild(cancelBtn)

	dialog.AddChild(nameLabel)
	dialog.AddChild(nameInput)
	dialog.AddChild(buttonsRow2)
	renameOverlay.AddChild(dialog)

	layerPanel.openRenameDialog = func(idx int, current string) {
		renameIdx = idx
		nameInput.SetText(current)
		nameInput.Focus(true)
		renameOverlay.GetWidget().Visibility = widget.Visibility_Show
	}

	// Wire the Rename button to open the rename dialog for the currently
	// selected layer.
	if renameBtn != nil {
		renameBtn.ClickedEvent.AddHandler(event.WrapHandler(func(args *widget.ButtonClickedEventArgs) {
			se := layerList.SelectedEntry()
			if se == nil {
				return
			}
			// Prefer the name stored in the LayerEntry; fall back to a default.
			if sel, ok := se.(LayerEntry); ok {
				name := sel.Name
				if name == "" {
					name = fmt.Sprintf("Layer %d", sel.Index)
				}
				if layerPanel.openRenameDialog != nil {
					layerPanel.openRenameDialog(sel.Index, name)
				}
				return
			}
			// No additional fallback; if SelectedEntry isn't a LayerEntry do nothing.
		}))
	}

	// Main grid container (placeholder)
	gridPanel := widget.NewContainer(
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.MinSize(800, 600),
		),
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	// save dialog and debug overlays removed

	// Root container: anchor layout
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	leftPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		StretchVertical:    true,
	}
	tilesetPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionEnd,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		StretchVertical:    true,
	}
	gridPanel.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
		StretchVertical:    true,
	}
	// Toolbar: top center
	toolbar.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
	}
	root.AddChild(gridPanel)
	root.AddChild(leftPanel)
	root.AddChild(tilesetPanel)
	root.AddChild(toolbar)
	root.AddChild(renameOverlay)

	// Ensure modal overlays stretch to cover the root and center their dialogs.
	if renameOverlay != nil && renameOverlay.GetWidget() != nil {
		renameOverlay.GetWidget().LayoutData = widget.AnchorLayoutData{
			HorizontalPosition: widget.AnchorLayoutPositionCenter,
			VerticalPosition:   widget.AnchorLayoutPositionCenter,
			StretchHorizontal:  true,
			StretchVertical:    true,
		}
	}
	// save dialog removed

	ui.Container = root
	if initialLayers != nil {
		layerPanel.SetLayers(initialLayers)
		layerPanel.SetSelected(initialLayerIndex)
	}
	layerPanel.SetAutotileButtonState(initialAutotileEnabled)

	return ui, &ToolBar{
		group:   group,
		buttons: toolButtons,
	}, layerPanel, fileNameInput, applyTileset, setTilesetSelection, setTilesetSelectionEnabled, transitionUI
}
