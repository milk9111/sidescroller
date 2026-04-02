package scenes

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"path/filepath"
	"strings"

	"github.com/ebitenui/ebitenui"
	euiimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/internal/savegame"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	startMenuButtonWidth    = 320
	startMenuButtonHeight   = 66
	startMenuSlotWidth      = 680
	startMenuSlotHeight     = 104
	startMenuContentOffsetY = -200
	startMenuSlideSpeed     = 0.12
	startMenuFadeSpeed      = 1.0 / 18.0
	startMenuSlots          = 4
	startMenuTitleMaxWidth  = common.BaseWidth - 200
	startMenuStickThreshold = 0.55
)

var startMenuBackground = color.NRGBA{R: 0x0b, G: 0x10, B: 0x16, A: 0xff}

type startMenuTheme struct {
	ButtonFace           textv2.Face
	SlotFace             textv2.Face
	LabelFace            textv2.Face
	ButtonText           *widget.ButtonTextColor
	PrimaryButtonImage   *widget.ButtonImage
	PrimaryFocusImage    *widget.ButtonImage
	SecondaryButtonImage *widget.ButtonImage
	SecondaryFocusImage  *widget.ButtonImage
	SlotButtonImage      *widget.ButtonImage
	SlotFocusImage       *widget.ButtonImage
	PanelBackground      *euiimage.NineSlice
	SubtlePanel          *euiimage.NineSlice
	TextColor            color.Color
	MutedTextColor       color.Color
	ErrorTextColor       color.Color
	ButtonPadding        *widget.Insets
	PanelPadding         *widget.Insets
}

type startMenuSlot struct {
	FileName string
	Summary  string
	Save     *savegame.File
}

type startMenuEntry struct {
	button     *widget.Button
	baseImage  *widget.ButtonImage
	focusImage *widget.ButtonImage
	onActivate func()
	disabled   bool
}

type StartMenuScene struct {
	config        *GameConfig
	theme         *startMenuTheme
	titleImage    *ebiten.Image
	mainUI        *ebitenui.UI
	loadUI        *ebitenui.UI
	mainSurface   *ebiten.Image
	loadSurface   *ebiten.Image
	slots         [startMenuSlots]startMenuSlot
	loadError     string
	slide         float64
	slideTarget   float64
	fadeAlpha     float64
	fadeTarget    string
	quitOnFade    bool
	mainEntries   []startMenuEntry
	loadEntries   []startMenuEntry
	mainFocus     int
	loadFocus     int
	axisUpHeld    bool
	axisDownHeld  bool
	axisLeftHeld  bool
	axisRightHeld bool
}

func NewStartMenuScene(config *GameConfig) (*StartMenuScene, error) {
	if config == nil {
		return nil, fmt.Errorf("start menu: nil game config")
	}

	titleImage, err := assets.LoadImage("title.png")
	if err != nil {
		return nil, fmt.Errorf("start menu: load title image: %w", err)
	}

	theme, err := newStartMenuTheme()
	if err != nil {
		return nil, err
	}

	scene := &StartMenuScene{
		config:      config,
		theme:       theme,
		titleImage:  clampGraphicWidth(titleImage, startMenuTitleMaxWidth),
		mainSurface: ebiten.NewImage(common.BaseWidth, common.BaseHeight),
		loadSurface: ebiten.NewImage(common.BaseWidth, common.BaseHeight),
	}
	if err := scene.refreshSlots(); err != nil {
		scene.loadError = err.Error()
	}
	if err := scene.buildUIs(); err != nil {
		return nil, err
	}

	return scene, nil
}

func (s *StartMenuScene) Update() (string, error) {
	if s.isFading() {
		s.fadeAlpha += startMenuFadeSpeed
		if s.fadeAlpha >= 1 {
			if s.quitOnFade {
				return "", ErrQuit
			}
			return s.fadeTarget, nil
		}
		return SceneStartMenu, nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if s.isLoadView() {
			s.slideTarget = 0
		} else {
			return "", ErrQuit
		}
	}

	if s.slide != s.slideTarget {
		s.slide = moveToward(s.slide, s.slideTarget, startMenuSlideSpeed)
		return SceneStartMenu, nil
	}

	if s.handleMenuNavigation() {
		return SceneStartMenu, nil
	}

	if s.isLoadView() {
		if s.loadUI != nil {
			s.loadUI.Update()
		}
	} else if s.mainUI != nil {
		s.mainUI.Update()
	}

	return SceneStartMenu, nil
}

func (s *StartMenuScene) Draw(screen *ebiten.Image) {
	screen.Fill(startMenuBackground)

	eased := smoothStep(s.slide)
	mainX := -eased * common.BaseWidth
	loadX := (1 - eased) * common.BaseWidth

	s.drawUI(screen, s.mainSurface, s.mainUI, mainX)
	s.drawUI(screen, s.loadSurface, s.loadUI, loadX)

	if s.fadeAlpha > 0 {
		overlay := ebiten.NewImage(1, 1)
		overlay.Fill(color.Black)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(common.BaseWidth, common.BaseHeight)
		op.ColorScale.ScaleAlpha(float32(minFloat(1, s.fadeAlpha)))
		screen.DrawImage(overlay, op)
	}
}

func (s *StartMenuScene) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return common.BaseWidth, common.BaseHeight
}

func (s *StartMenuScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return common.BaseWidth, common.BaseHeight
}

func (s *StartMenuScene) buildUIs() error {
	mainUI, err := s.buildMainUI()
	if err != nil {
		return err
	}
	loadUI, err := s.buildLoadUI()
	if err != nil {
		return err
	}
	s.mainUI = mainUI
	s.loadUI = loadUI
	return nil
}

func (s *StartMenuScene) buildMainUI() (*ebitenui.UI, error) {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	content := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(28),
		)),
	)
	content.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	content.GetWidget().MinHeight = max(1, common.BaseHeight+startMenuContentOffsetY*2)

	title := widget.NewGraphic(
		widget.GraphicOpts.Image(s.titleImage),
		widget.GraphicOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)

	buttons := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(14),
		)),
	)
	buttons.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionCenter}

	s.mainEntries = s.mainEntries[:0]
	startAction := func() {
		s.config.LoadedSave = nil
		s.config.InitialFadeIn = true
		s.beginFade(SceneIntro, false)
	}
	startButton := s.newMenuButton("Start", s.theme.PrimaryButtonImage, startAction)
	s.mainEntries = append(s.mainEntries, startMenuEntry{button: startButton, baseImage: s.theme.PrimaryButtonImage, focusImage: s.theme.PrimaryFocusImage, onActivate: startAction})
	buttons.AddChild(startButton)

	loadAction := func() {
		if err := s.refreshSlots(); err != nil {
			s.loadError = err.Error()
		} else {
			s.loadError = ""
		}
		if err := s.rebuildLoadUI(); err != nil {
			s.loadError = err.Error()
		}
		s.slideTarget = 1
	}
	loadButton := s.newMenuButton("Load", s.theme.SecondaryButtonImage, loadAction)
	s.mainEntries = append(s.mainEntries, startMenuEntry{button: loadButton, baseImage: s.theme.SecondaryButtonImage, focusImage: s.theme.SecondaryFocusImage, onActivate: loadAction})
	buttons.AddChild(loadButton)

	exitAction := func() {
		s.beginFade("", true)
	}
	exitButton := s.newMenuButton("Exit", s.theme.SecondaryButtonImage, exitAction)
	s.mainEntries = append(s.mainEntries, startMenuEntry{button: exitButton, baseImage: s.theme.SecondaryButtonImage, focusImage: s.theme.SecondaryFocusImage, onActivate: exitAction})
	buttons.AddChild(exitButton)

	content.AddChild(title)
	content.AddChild(buttons)
	root.AddChild(content)
	s.mainFocus = clampMenuFocus(s.mainEntries, s.mainFocus)
	s.applyFocusState(s.mainEntries, s.mainFocus)

	return &ebitenui.UI{Container: root}, nil
}

func (s *StartMenuScene) buildLoadUI() (*ebitenui.UI, error) {
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	content := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(18),
		)),
	)
	content.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionCenter,
		VerticalPosition:   widget.AnchorLayoutPositionCenter,
	}
	content.GetWidget().MinHeight = max(1, common.BaseHeight+startMenuContentOffsetY*2)

	title := widget.NewGraphic(
		widget.GraphicOpts.Image(s.titleImage),
		widget.GraphicOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)
	content.AddChild(title)

	header := widget.NewText(
		widget.TextOpts.Text("Select Save", &s.theme.LabelFace, s.theme.TextColor),
		widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
	)
	content.AddChild(header)

	panel := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(s.theme.PanelBackground),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(12),
			widget.RowLayoutOpts.Padding(s.theme.PanelPadding),
		)),
	)
	panel.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionCenter}

	s.loadEntries = s.loadEntries[:0]
	for i := range s.slots {
		slot := s.slots[i]
		button := widget.NewButton(
			widget.ButtonOpts.Image(s.theme.SlotButtonImage),
			widget.ButtonOpts.Text(slot.Summary, &s.theme.SlotFace, s.theme.ButtonText),
			widget.ButtonOpts.TextPadding(&widget.Insets{Left: 20, Right: 20, Top: 16, Bottom: 16}),
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(startMenuSlotWidth, startMenuSlotHeight)),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
				s.loadSlot(slot)
			}),
		)
		if slot.Save == nil {
			button.GetWidget().Disabled = true
		}
		s.loadEntries = append(s.loadEntries, startMenuEntry{
			button:     button,
			baseImage:  s.theme.SlotButtonImage,
			focusImage: s.theme.SlotFocusImage,
			onActivate: func() { s.loadSlot(slot) },
			disabled:   slot.Save == nil,
		})
		panel.AddChild(button)
	}

	content.AddChild(panel)

	if strings.TrimSpace(s.loadError) != "" {
		errorText := widget.NewText(
			widget.TextOpts.Text(s.loadError, &s.theme.LabelFace, s.theme.ErrorTextColor),
			widget.TextOpts.Position(widget.TextPositionCenter, widget.TextPositionCenter),
			widget.TextOpts.MaxWidth(common.BaseWidth-220),
			widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter})),
		)
		content.AddChild(errorText)
	}

	backButton := s.newMenuButton("Back", s.theme.SecondaryButtonImage, func() {
		s.slideTarget = 0
	})
	s.loadEntries = append(s.loadEntries, startMenuEntry{button: backButton, baseImage: s.theme.SecondaryButtonImage, focusImage: s.theme.SecondaryFocusImage, onActivate: func() {
		s.slideTarget = 0
	}})
	content.AddChild(backButton)
	root.AddChild(content)
	s.loadFocus = clampMenuFocus(s.loadEntries, s.loadFocus)
	s.applyFocusState(s.loadEntries, s.loadFocus)

	return &ebitenui.UI{Container: root}, nil
}

func (s *StartMenuScene) rebuildLoadUI() error {
	loadUI, err := s.buildLoadUI()
	if err != nil {
		return err
	}
	s.loadUI = loadUI
	return nil
}

func (s *StartMenuScene) newMenuButton(label string, image *widget.ButtonImage, onClick func()) *widget.Button {
	button := widget.NewButton(
		widget.ButtonOpts.Image(image),
		widget.ButtonOpts.Text(label, &s.theme.ButtonFace, s.theme.ButtonText),
		widget.ButtonOpts.TextPadding(s.theme.ButtonPadding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.MinSize(startMenuButtonWidth, startMenuButtonHeight)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if onClick != nil {
				onClick()
			}
		}),
	)
	return button
}

func (s *StartMenuScene) loadSlot(slot startMenuSlot) {
	store, err := savegame.NewStore(slot.FileName, log.Printf)
	if err != nil {
		s.loadError = err.Error()
		_ = s.rebuildLoadUI()
		return
	}

	loadedSave, err := store.Load()
	if err != nil {
		s.loadError = err.Error()
		_ = s.rebuildLoadUI()
		return
	}

	s.config.SaveStore = store
	s.config.LoadedSave = loadedSave
	s.config.LevelName = loadedSave.Level
	s.config.InitialFadeIn = true
	s.beginFade(SceneGame, false)
}

func (s *StartMenuScene) refreshSlots() error {
	listed, err := savegame.ListSlots(startMenuSlots)
	if err != nil {
		for i := range s.slots {
			s.slots[i] = emptyStartMenuSlot(i)
		}
		return fmt.Errorf("load save slots: %w", err)
	}

	for i := range s.slots {
		s.slots[i] = emptyStartMenuSlot(i)
	}
	for i := range listed {
		s.slots[i] = startMenuSlot{
			FileName: listed[i].FileName,
			Summary:  formatSlotSummary(listed[i].FileName, listed[i].Snapshot),
			Save:     listed[i].Snapshot,
		}
	}
	return nil
}

func (s *StartMenuScene) beginFade(target string, quit bool) {
	s.fadeTarget = target
	s.quitOnFade = quit
	s.fadeAlpha = maxFloat(s.fadeAlpha, startMenuFadeSpeed)
}

func (s *StartMenuScene) handleMenuNavigation() bool {
	entries, focus := s.activeEntries()
	if len(entries) == 0 {
		return false
	}

	moved := false
	if menuNavigatePreviousPressed(&s.axisUpHeld, &s.axisLeftHeld) {
		focus = nextMenuFocus(entries, focus, -1)
		moved = true
	}
	if menuNavigateNextPressed(&s.axisDownHeld, &s.axisRightHeld) {
		focus = nextMenuFocus(entries, focus, 1)
		moved = true
	}
	if moved {
		s.setActiveFocus(focus)
		return true
	}

	if menuActivatePressed() {
		entry := entries[focus]
		if !entry.disabled && entry.onActivate != nil {
			entry.onActivate()
			return true
		}
	}

	return false
}

func (s *StartMenuScene) activeEntries() ([]startMenuEntry, int) {
	if s.isLoadView() {
		if len(s.loadEntries) == 0 {
			return nil, -1
		}
		return s.loadEntries, clampMenuFocus(s.loadEntries, s.loadFocus)
	}
	if len(s.mainEntries) == 0 {
		return nil, -1
	}
	return s.mainEntries, clampMenuFocus(s.mainEntries, s.mainFocus)
}

func (s *StartMenuScene) setActiveFocus(index int) {
	if s.isLoadView() {
		s.loadFocus = clampMenuFocus(s.loadEntries, index)
		s.applyFocusState(s.loadEntries, s.loadFocus)
		return
	}
	s.mainFocus = clampMenuFocus(s.mainEntries, index)
	s.applyFocusState(s.mainEntries, s.mainFocus)
}

func (s *StartMenuScene) applyFocusState(entries []startMenuEntry, focused int) {
	for index := range entries {
		entry := entries[index]
		if entry.button == nil {
			continue
		}
		if index == focused && !entry.disabled && entry.focusImage != nil {
			entry.button.SetImage(entry.focusImage)
		} else if entry.baseImage != nil {
			entry.button.SetImage(entry.baseImage)
		}
	}
}

func (s *StartMenuScene) drawUI(screen, surface *ebiten.Image, ui *ebitenui.UI, offsetX float64) {
	if screen == nil || surface == nil || ui == nil {
		return
	}
	surface.Clear()
	ui.Draw(surface)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(offsetX, 0)
	screen.DrawImage(surface, op)
}

func (s *StartMenuScene) isLoadView() bool {
	return s.slideTarget > 0 || s.slide > 0
}

func (s *StartMenuScene) isFading() bool {
	return s.fadeAlpha > 0
}

func newStartMenuTheme() (*startMenuTheme, error) {
	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, fmt.Errorf("start menu: load font: %w", err)
	}

	return &startMenuTheme{
		ButtonFace: textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 28}),
		SlotFace:   textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 18}),
		LabelFace:  textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 22}),
		ButtonText: &widget.ButtonTextColor{
			Idle:     color.NRGBA{R: 239, G: 243, B: 248, A: 255},
			Disabled: color.NRGBA{R: 132, G: 141, B: 156, A: 255},
			Hover:    color.NRGBA{R: 255, G: 255, B: 255, A: 255},
			Pressed:  color.NRGBA{R: 255, G: 255, B: 255, A: 255},
		},
		PrimaryButtonImage: &widget.ButtonImage{
			Idle:         euiimage.NewNineSliceColor(color.NRGBA{R: 31, G: 92, B: 125, A: 255}),
			Hover:        euiimage.NewNineSliceColor(color.NRGBA{R: 43, G: 113, B: 149, A: 255}),
			Pressed:      euiimage.NewNineSliceColor(color.NRGBA{R: 24, G: 73, B: 98, A: 255}),
			PressedHover: euiimage.NewNineSliceColor(color.NRGBA{R: 24, G: 73, B: 98, A: 255}),
			Disabled:     euiimage.NewNineSliceColor(color.NRGBA{R: 28, G: 38, B: 48, A: 255}),
		},
		PrimaryFocusImage: &widget.ButtonImage{
			Idle:         euiimage.NewNineSliceColor(color.NRGBA{R: 52, G: 129, B: 166, A: 255}),
			Hover:        euiimage.NewNineSliceColor(color.NRGBA{R: 63, G: 145, B: 184, A: 255}),
			Pressed:      euiimage.NewNineSliceColor(color.NRGBA{R: 40, G: 104, B: 135, A: 255}),
			PressedHover: euiimage.NewNineSliceColor(color.NRGBA{R: 40, G: 104, B: 135, A: 255}),
			Disabled:     euiimage.NewNineSliceColor(color.NRGBA{R: 28, G: 38, B: 48, A: 255}),
		},
		SecondaryButtonImage: &widget.ButtonImage{
			Idle:         euiimage.NewNineSliceColor(color.NRGBA{R: 26, G: 34, B: 43, A: 255}),
			Hover:        euiimage.NewNineSliceColor(color.NRGBA{R: 37, G: 48, B: 60, A: 255}),
			Pressed:      euiimage.NewNineSliceColor(color.NRGBA{R: 20, G: 27, B: 34, A: 255}),
			PressedHover: euiimage.NewNineSliceColor(color.NRGBA{R: 20, G: 27, B: 34, A: 255}),
			Disabled:     euiimage.NewNineSliceColor(color.NRGBA{R: 20, G: 27, B: 34, A: 255}),
		},
		SecondaryFocusImage: &widget.ButtonImage{
			Idle:         euiimage.NewNineSliceColor(color.NRGBA{R: 47, G: 63, B: 79, A: 255}),
			Hover:        euiimage.NewNineSliceColor(color.NRGBA{R: 58, G: 75, B: 93, A: 255}),
			Pressed:      euiimage.NewNineSliceColor(color.NRGBA{R: 37, G: 49, B: 61, A: 255}),
			PressedHover: euiimage.NewNineSliceColor(color.NRGBA{R: 37, G: 49, B: 61, A: 255}),
			Disabled:     euiimage.NewNineSliceColor(color.NRGBA{R: 20, G: 27, B: 34, A: 255}),
		},
		SlotButtonImage: &widget.ButtonImage{
			Idle:         euiimage.NewNineSliceColor(color.NRGBA{R: 18, G: 25, B: 33, A: 255}),
			Hover:        euiimage.NewNineSliceColor(color.NRGBA{R: 24, G: 34, B: 45, A: 255}),
			Pressed:      euiimage.NewNineSliceColor(color.NRGBA{R: 16, G: 22, B: 29, A: 255}),
			PressedHover: euiimage.NewNineSliceColor(color.NRGBA{R: 16, G: 22, B: 29, A: 255}),
			Disabled:     euiimage.NewNineSliceColor(color.NRGBA{R: 14, G: 18, B: 24, A: 220}),
		},
		SlotFocusImage: &widget.ButtonImage{
			Idle:         euiimage.NewNineSliceColor(color.NRGBA{R: 32, G: 52, B: 70, A: 255}),
			Hover:        euiimage.NewNineSliceColor(color.NRGBA{R: 38, G: 60, B: 80, A: 255}),
			Pressed:      euiimage.NewNineSliceColor(color.NRGBA{R: 27, G: 44, B: 58, A: 255}),
			PressedHover: euiimage.NewNineSliceColor(color.NRGBA{R: 27, G: 44, B: 58, A: 255}),
			Disabled:     euiimage.NewNineSliceColor(color.NRGBA{R: 14, G: 18, B: 24, A: 220}),
		},
		PanelBackground: euiimage.NewNineSliceColor(color.NRGBA{R: 11, G: 17, B: 23, A: 236}),
		SubtlePanel:     euiimage.NewNineSliceColor(color.NRGBA{R: 14, G: 20, B: 27, A: 210}),
		TextColor:       color.NRGBA{R: 231, G: 237, B: 244, A: 255},
		MutedTextColor:  color.NRGBA{R: 150, G: 164, B: 179, A: 255},
		ErrorTextColor:  color.NRGBA{R: 228, G: 127, B: 121, A: 255},
		ButtonPadding:   &widget.Insets{Left: 18, Right: 18, Top: 10, Bottom: 10},
		PanelPadding:    &widget.Insets{Left: 20, Right: 20, Top: 20, Bottom: 20},
	}, nil
}

func clampGraphicWidth(img *ebiten.Image, maxWidth int) *ebiten.Image {
	if img == nil || maxWidth <= 0 {
		return img
	}
	bounds := img.Bounds()
	if bounds.Dx() <= maxWidth {
		return img
	}
	scale := float64(maxWidth) / float64(bounds.Dx())
	resized := ebiten.NewImage(maxWidth, max(1, int(float64(bounds.Dy())*scale)))
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	resized.DrawImage(img, op)
	return resized
}

func emptyStartMenuSlot(index int) startMenuSlot {
	return startMenuSlot{
		Summary: fmt.Sprintf("Slot %d\nEmpty\nNo save file found in the save directory.", index+1),
	}
}

func formatSlotSummary(fileName string, save *savegame.File) string {
	if save == nil {
		return fmt.Sprintf("%s\nUnreadable save", fileName)
	}

	savedAt := "Unknown save time"
	if !save.SavedAt.IsZero() {
		savedAt = save.SavedAt.Local().Format("2006-01-02 15:04")
	}
	levelName := strings.TrimSuffix(filepath.Base(strings.TrimSpace(save.Level)), filepath.Ext(strings.TrimSpace(save.Level)))
	if levelName == "" {
		levelName = "Unknown level"
	}

	return fmt.Sprintf(
		"%s\nSaved %s  |  Level %s\nHealth %d/%d  |  Gear %d  |  Items %d\nAbilities %s",
		fileName,
		savedAt,
		levelName,
		save.Player.Health.Current,
		save.Player.Health.Initial,
		save.Player.GearCount,
		len(save.Player.Inventory),
		formatAbilities(save.Player.Abilities),
	)
}

func formatAbilities(abilities savegame.AbilitiesState) string {
	enabled := make([]string, 0, 4)
	if abilities.Anchor {
		enabled = append(enabled, "anchor")
	}
	if abilities.DoubleJump {
		enabled = append(enabled, "double jump")
	}
	if abilities.WallGrab {
		enabled = append(enabled, "wall grab")
	}
	if abilities.Heal {
		enabled = append(enabled, "heal")
	}
	if len(enabled) == 0 {
		return "none"
	}
	return strings.Join(enabled, ", ")
}

func smoothStep(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 1
	}
	return value * value * (3 - 2*value)
}

func moveToward(current, target, step float64) float64 {
	if current < target {
		return minFloat(target, current+step)
	}
	if current > target {
		return maxFloat(target, current-step)
	}
	return current
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func clampMenuFocus(entries []startMenuEntry, index int) int {
	if len(entries) == 0 {
		return -1
	}
	if index < 0 || index >= len(entries) || entries[index].disabled {
		for candidate := range entries {
			if !entries[candidate].disabled {
				return candidate
			}
		}
		return 0
	}
	return index
}

func nextMenuFocus(entries []startMenuEntry, current, delta int) int {
	if len(entries) == 0 {
		return -1
	}
	if delta == 0 {
		return clampMenuFocus(entries, current)
	}
	current = clampMenuFocus(entries, current)
	for step := 1; step <= len(entries); step++ {
		candidate := (current + step*delta + len(entries)*4) % len(entries)
		if !entries[candidate].disabled {
			return candidate
		}
	}
	return current
}

func menuActivatePressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		return true
	}
	if gamepads := ebiten.GamepadIDs(); len(gamepads) > 0 {
		id := gamepads[0]
		return inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) ||
			inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightLeft)
	}
	return false
}

func menuNavigatePreviousPressed(verticalHeld, horizontalHeld *bool) bool {
	pressed := inpututil.IsKeyJustPressed(ebiten.KeyW) ||
		inpututil.IsKeyJustPressed(ebiten.KeyA) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft)

	verticalPressed := menuGamepadVerticalDirection() < 0
	horizontalPressed := menuGamepadHorizontalDirection() < 0
	if verticalPressed && !*verticalHeld {
		*verticalHeld = true
		pressed = true
	}
	if !verticalPressed {
		*verticalHeld = false
	}
	if horizontalPressed && !*horizontalHeld {
		*horizontalHeld = true
		pressed = true
	}
	if !horizontalPressed {
		*horizontalHeld = false
	}
	return pressed
}

func menuNavigateNextPressed(verticalHeld, horizontalHeld *bool) bool {
	pressed := inpututil.IsKeyJustPressed(ebiten.KeyS) ||
		inpututil.IsKeyJustPressed(ebiten.KeyD) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowRight)

	verticalPressed := menuGamepadVerticalDirection() > 0
	horizontalPressed := menuGamepadHorizontalDirection() > 0
	if verticalPressed && !*verticalHeld {
		*verticalHeld = true
		pressed = true
	}
	if !verticalPressed {
		*verticalHeld = false
	}
	if horizontalPressed && !*horizontalHeld {
		*horizontalHeld = true
		pressed = true
	}
	if !horizontalPressed {
		*horizontalHeld = false
	}
	return pressed
}

func menuGamepadVerticalDirection() int {
	if gamepads := ebiten.GamepadIDs(); len(gamepads) > 0 {
		id := gamepads[0]
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			return -1
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			return 1
		}
		leftY := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickVertical)
		if leftY <= -startMenuStickThreshold {
			return -1
		}
		if leftY >= startMenuStickThreshold {
			return 1
		}
	}
	return 0
}

func menuGamepadHorizontalDirection() int {
	if gamepads := ebiten.GamepadIDs(); len(gamepads) > 0 {
		id := gamepads[0]
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			return -1
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			return 1
		}
		leftX := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
		if leftX <= -startMenuStickThreshold {
			return -1
		}
		if leftX >= startMenuStickThreshold {
			return 1
		}
	}
	return 0
}
