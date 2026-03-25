package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"gopkg.in/yaml.v3"

	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	editorcomponents "github.com/milk9111/sidescroller/cmd/editor/ui/components"
	"github.com/milk9111/sidescroller/prefabs"
)

const (
	inspectorWidth   = 340
	toolbarHeight    = 64
	previewPadding   = 20
	previewGridSize  = 24
	previewOriginY   = 0.72
	previewOriginMin = 72
)

var (
	ErrQuit = errors.New("quit")
)

type AppConfig struct {
	WorkspaceRoot string
	AssetDir      string
	PrefabDir     string
	File          string
	Assets        []editorio.AssetInfo
}

type emitterForm struct {
	Name           string
	TotalParticles string
	Lifetime       string
	Image          string
	Color          string
	Burst          bool
	Continuous     bool
	HasGravity     bool
	ScaleX         string
	ScaleY         string
}

type validatedEmitter struct {
	Name string
	Spec prefabs.ParticleEmitterComponentSpec
}

type App struct {
	workspaceRoot string
	prefabDir     string
	assets        []editorio.AssetInfo

	theme *editorcomponents.Theme
	ui    *ebitenui.UI

	form emitterForm

	status           string
	validation       string
	previewStateText string
	dirty            bool

	nameInput           *widget.TextInput
	totalParticlesInput *widget.TextInput
	lifetimeInput       *widget.TextInput
	imageInput          *widget.TextInput
	colorInput          *widget.TextInput
	scaleXInput         *widget.TextInput
	scaleYInput         *widget.TextInput
	burstButton         *widget.Button
	continuousButton    *widget.Button
	gravityButton       *widget.Button
	playButton          *widget.Button
	pauseButton         *widget.Button
	stopButton          *widget.Button
	restartButton       *widget.Button
	saveButton          *widget.Button
	statusText          *widget.Text
	validationText      *widget.Text
	summaryText         *widget.Text
	previewModeText     *widget.Text

	preview    *particlePreview
	screenSize image.Point
}

func NewApp(cfg AppConfig) (*App, error) {
	theme, err := editorcomponents.NewTheme()
	if err != nil {
		return nil, err
	}

	form := defaultEmitterForm()
	status := fmt.Sprintf("Ready. %d assets scanned.", len(cfg.Assets))
	if strings.TrimSpace(cfg.File) != "" {
		loaded, loadErr := loadEmitterPrefab(cfg.WorkspaceRoot, cfg.PrefabDir, cfg.File)
		if loadErr != nil {
			return nil, loadErr
		}
		form = loaded.form
		status = fmt.Sprintf("Loaded %s/%s", cfg.PrefabDir, loaded.normalizedPath)
	}

	app := &App{
		workspaceRoot:    cfg.WorkspaceRoot,
		prefabDir:        cfg.PrefabDir,
		assets:           append([]editorio.AssetInfo(nil), cfg.Assets...),
		theme:            theme,
		preview:          newParticlePreview(),
		form:             form,
		status:           status,
		previewStateText: "Playing",
	}

	ui, err := app.buildUI()
	if err != nil {
		return nil, err
	}
	app.ui = ui
	app.syncWidgets()
	app.revalidate(true)
	app.dirty = false
	app.syncStatus()
	return app, nil
}

type loadedEmitterPrefab struct {
	form           emitterForm
	normalizedPath string
}

func defaultEmitterForm() emitterForm {
	return emitterForm{
		Name:           "particle_emitter",
		TotalParticles: "100",
		Lifetime:       "30",
		Image:          "basic_particle.png",
		Color:          "#FFFFFF",
		Burst:          true,
		Continuous:     false,
		HasGravity:     false,
		ScaleX:         "1",
		ScaleY:         "1",
	}
}

func loadEmitterPrefab(workspaceRoot, prefabDir, target string) (loadedEmitterPrefab, error) {
	normalized, err := editorio.NormalizePrefabTarget(target)
	if err != nil {
		return loadedEmitterPrefab{}, err
	}
	path := filepath.Join(workspaceRoot, prefabDir, normalized)
	data, err := os.ReadFile(path)
	if err != nil {
		return loadedEmitterPrefab{}, fmt.Errorf("read prefab %q: %w", path, err)
	}
	var spec prefabs.EntityBuildSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return loadedEmitterPrefab{}, fmt.Errorf("decode prefab %q: %w", normalized, err)
	}
	if spec.Components == nil {
		return loadedEmitterPrefab{}, fmt.Errorf("prefab %q has no components", normalized)
	}
	rawEmitter, ok := spec.Components["particle_emitter"]
	if !ok {
		return loadedEmitterPrefab{}, fmt.Errorf("prefab %q does not contain a particle_emitter component", normalized)
	}
	emitter, err := prefabs.DecodeComponentSpec[prefabs.ParticleEmitterComponentSpec](rawEmitter)
	if err != nil {
		return loadedEmitterPrefab{}, fmt.Errorf("decode particle_emitter for %q: %w", normalized, err)
	}
	name := strings.TrimSuffix(normalized, filepath.Ext(normalized))
	return loadedEmitterPrefab{
		form: emitterForm{
			Name:           name,
			TotalParticles: strconv.Itoa(emitter.TotalParticles),
			Lifetime:       strconv.Itoa(emitter.Lifetime),
			Image:          emitter.Image,
			Color:          emitter.Color,
			Burst:          emitter.Burst,
			Continuous:     emitter.Continuous,
			HasGravity:     emitter.HasGravity,
			ScaleX:         fmt.Sprintf("%g", emitter.Scale.X),
			ScaleY:         fmt.Sprintf("%g", emitter.Scale.Y),
		},
		normalizedPath: normalized,
	}, nil
}

func (a *App) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		return ErrQuit
	}
	if isSaveShortcutPressed() {
		a.saveCurrentEmitter()
	}
	if a.ui != nil {
		a.ui.Update()
	}
	originX, originY := a.previewOrigin()
	a.preview.Update(originX, originY)
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	a.screenSize = image.Pt(screen.Bounds().Dx(), screen.Bounds().Dy())
	screen.Fill(color.NRGBA{R: 10, G: 12, B: 18, A: 255})
	a.drawPreview(screen)
	if a.ui != nil {
		a.ui.Draw(screen)
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func (a *App) buildUI() (*ebitenui.UI, error) {
	var app *App
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)

	toolbar := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(a.theme.ToolbarBackground),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(10),
			widget.RowLayoutOpts.Padding(a.theme.PanelPadding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionCenter,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchHorizontal:  true,
			}),
		),
	)
	toolbar.GetWidget().MinHeight = toolbarHeight

	toolbar.AddChild(newTextLabel(a.theme, "Particles", &a.theme.TitleFace, a.theme.TextColor))

	playButton := newActionButton(a.theme, "Play", func() {
		if app != nil {
			app.preview.Play()
			app.previewStateText = "Playing"
			app.syncStatus()
		}
	})
	pauseButton := newActionButton(a.theme, "Pause", func() {
		if app != nil {
			app.preview.Pause()
			app.previewStateText = "Paused"
			app.syncStatus()
		}
	})
	stopButton := newActionButton(a.theme, "Stop", func() {
		if app != nil {
			app.preview.Stop()
			app.previewStateText = "Stopped"
			app.syncStatus()
		}
	})
	restartButton := newActionButton(a.theme, "Restart", func() {
		if app != nil {
			app.preview.Restart()
			app.previewStateText = "Playing"
			app.syncStatus()
		}
	})
	saveButton := newActionButton(a.theme, "Save", func() {
		if app != nil {
			app.saveCurrentEmitter()
		}
	})
	toolbar.AddChild(playButton)
	toolbar.AddChild(pauseButton)
	toolbar.AddChild(stopButton)
	toolbar.AddChild(restartButton)
	toolbar.AddChild(saveButton)

	previewModeText := newTextLabel(a.theme, "Preview: Playing", &a.theme.Face, a.theme.MutedTextColor)
	statusText := newTextLabel(a.theme, a.status, &a.theme.Face, a.theme.MutedTextColor)
	toolbar.AddChild(previewModeText)
	toolbar.AddChild(statusText)

	inspector := widget.NewContainer(
		widget.ContainerOpts.BackgroundImage(a.theme.PanelBackground),
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(8),
			widget.RowLayoutOpts.Padding(a.theme.PanelPadding),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchVertical:    true,
				Padding:            &widget.Insets{Top: toolbarHeight},
			}),
		),
	)
	inspector.GetWidget().MinWidth = inspectorWidth

	inspector.AddChild(newTextLabel(a.theme, "Inspector", &a.theme.TitleFace, a.theme.TextColor))
	summaryText := newTextLabel(a.theme, "Every field below maps directly to the particle_emitter component spec.", &a.theme.Face, a.theme.MutedTextColor)
	validationText := newTextLabel(a.theme, "", &a.theme.Face, color.NRGBA{R: 255, G: 154, B: 154, A: 255})
	inspector.AddChild(summaryText)
	inspector.AddChild(validationText)

	nameInput := newLabeledInput(inspector, a.theme, "Emitter Name", "File name in prefabs/", func(value string) {
		if app != nil {
			app.form.Name = value
			app.dirty = true
			app.revalidate(false)
		}
	})
	totalInput := newLabeledInput(inspector, a.theme, "Total Particles", "Positive integer", func(value string) {
		if app != nil {
			app.form.TotalParticles = value
			app.dirty = true
			app.revalidate(true)
		}
	})
	lifetimeInput := newLabeledInput(inspector, a.theme, "Lifetime", "Frames", func(value string) {
		if app != nil {
			app.form.Lifetime = value
			app.dirty = true
			app.revalidate(true)
		}
	})
	imageInput := newLabeledInput(inspector, a.theme, "Image", "Asset path relative to assets/", func(value string) {
		if app != nil {
			app.form.Image = value
			app.dirty = true
			app.revalidate(true)
		}
	})
	colorInput := newLabeledInput(inspector, a.theme, "Color", "#RRGGBB or #RRGGBBAA", func(value string) {
		if app != nil {
			app.form.Color = value
			app.dirty = true
			app.revalidate(true)
		}
	})

	scaleXInput, scaleYInput := newInlineTwoInputs(inspector, a.theme, "Scale", "X", "Y", func(v string) {
		if app != nil {
			app.form.ScaleX = v
			app.dirty = true
			app.revalidate(true)
		}
	}, func(v string) {
		if app != nil {
			app.form.ScaleY = v
			app.dirty = true
			app.revalidate(true)
		}
	})

	burstButton := newToggleButton(a.theme, func() {
		if app != nil {
			app.form.Burst = !app.form.Burst
			app.dirty = true
			app.revalidate(true)
		}
	})
	continuousButton := newToggleButton(a.theme, func() {
		if app != nil {
			app.form.Continuous = !app.form.Continuous
			app.dirty = true
			app.revalidate(true)
		}
	})
	gravityButton := newToggleButton(a.theme, func() {
		if app != nil {
			app.form.HasGravity = !app.form.HasGravity
			app.dirty = true
			app.revalidate(true)
		}
	})
	inspector.AddChild(burstButton)
	inspector.AddChild(continuousButton)
	inspector.AddChild(gravityButton)
	inspector.AddChild(newTextLabel(a.theme, fmt.Sprintf("Assets scanned: %d", len(a.assets)), &a.theme.Face, a.theme.MutedTextColor))
	inspector.AddChild(newTextLabel(a.theme, "Ctrl+S saves the current emitter prefab.", &a.theme.Face, a.theme.MutedTextColor))

	root.AddChild(toolbar)
	root.AddChild(inspector)

	app = a
	a.nameInput = nameInput
	a.totalParticlesInput = totalInput
	a.lifetimeInput = lifetimeInput
	a.imageInput = imageInput
	a.colorInput = colorInput
	a.burstButton = burstButton
	a.scaleXInput = scaleXInput
	a.scaleYInput = scaleYInput
	a.continuousButton = continuousButton
	a.gravityButton = gravityButton
	a.playButton = playButton
	a.pauseButton = pauseButton
	a.stopButton = stopButton
	a.restartButton = restartButton
	a.saveButton = saveButton
	a.statusText = statusText
	a.validationText = validationText
	a.summaryText = summaryText
	a.previewModeText = previewModeText

	return &ebitenui.UI{Container: root}, nil
}

func (a *App) drawPreview(screen *ebiten.Image) {
	rect := a.previewRect()
	if rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}

	vector.DrawFilledRect(screen, float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()), color.NRGBA{R: 18, G: 21, B: 29, A: 255}, true)

	for x := rect.Min.X; x < rect.Max.X; x += previewGridSize {
		gridColor := color.NRGBA{R: 29, G: 34, B: 45, A: 255}
		vector.StrokeLine(screen, float32(x), float32(rect.Min.Y), float32(x), float32(rect.Max.Y), 1, gridColor, true)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y += previewGridSize {
		gridColor := color.NRGBA{R: 29, G: 34, B: 45, A: 255}
		vector.StrokeLine(screen, float32(rect.Min.X), float32(y), float32(rect.Max.X), float32(y), 1, gridColor, true)
	}

	originX, originY := a.previewOrigin()
	vector.StrokeLine(screen, float32(originX-18), float32(originY), float32(originX+18), float32(originY), 2, color.NRGBA{R: 76, G: 120, B: 255, A: 255}, true)
	vector.StrokeLine(screen, float32(originX), float32(originY-18), float32(originX), float32(originY+18), 2, color.NRGBA{R: 76, G: 120, B: 255, A: 255}, true)
	a.preview.Draw(screen, rect, originX, originY)

	drawText(screen, "Live Preview", a.theme.TitleFace, float64(rect.Min.X+16), float64(rect.Min.Y+28), a.theme.TextColor)
	drawText(screen, fmt.Sprintf("%dx%d preview area", rect.Dx(), rect.Dy()), a.theme.Face, float64(rect.Min.X+16), float64(rect.Min.Y+52), a.theme.MutedTextColor)
	if a.validation != "" {
		drawText(screen, "Preview holds the last valid emitter until the form validates again.", a.theme.Face, float64(rect.Min.X+16), float64(rect.Max.Y-20), color.NRGBA{R: 255, G: 188, B: 122, A: 255})
	}
}

func (a *App) previewRect() image.Rectangle {
	left := inspectorWidth + previewPadding
	top := toolbarHeight + previewPadding
	right := a.screenSize.X - previewPadding
	bottom := a.screenSize.Y - previewPadding
	if right < left {
		right = left
	}
	if bottom < top {
		bottom = top
	}
	return image.Rect(left, top, right, bottom)
}

func (a *App) previewOrigin() (float64, float64) {
	rect := a.previewRect()
	originX := float64(rect.Min.X + rect.Dx()/2)
	originY := float64(rect.Min.Y) + float64(rect.Dy())*previewOriginY
	minY := float64(rect.Min.Y + previewOriginMin)
	if originY < minY {
		originY = minY
	}
	return originX, originY
}

func (a *App) syncWidgets() {
	if a.nameInput != nil {
		a.nameInput.SetText(a.form.Name)
	}
	if a.totalParticlesInput != nil {
		a.totalParticlesInput.SetText(a.form.TotalParticles)
	}
	if a.lifetimeInput != nil {
		a.lifetimeInput.SetText(a.form.Lifetime)
	}
	if a.imageInput != nil {
		a.imageInput.SetText(a.form.Image)
	}
	if a.colorInput != nil {
		a.colorInput.SetText(a.form.Color)
	}
	if a.scaleXInput != nil {
		a.scaleXInput.SetText(a.form.ScaleX)
	}
	if a.scaleYInput != nil {
		a.scaleYInput.SetText(a.form.ScaleY)
	}
	a.syncToggleButtons()
	a.syncStatus()
}

func (a *App) syncToggleButtons() {
	setToggleState(a.burstButton, a.theme, a.form.Burst, "Burst")
	setToggleState(a.continuousButton, a.theme, a.form.Continuous, "Continuous")
	setToggleState(a.gravityButton, a.theme, a.form.HasGravity, "Gravity")
}

func (a *App) syncStatus() {
	if a.statusText != nil {
		status := a.status
		if a.dirty {
			status += "  Unsaved changes."
		}
		a.statusText.Label = status
	}
	if a.validationText != nil {
		a.validationText.Label = a.validation
	}
	if a.previewModeText != nil {
		a.previewModeText.Label = "Preview: " + a.previewStateText
	}
	if a.summaryText != nil {
		a.summaryText.Label = fmt.Sprintf("Saving writes %s/<name>.yaml with transform and particle_emitter components.", a.prefabDir)
	}
	if a.saveButton != nil {
		a.saveButton.GetWidget().Disabled = strings.TrimSpace(a.validation) != ""
	}

	// Keep toggle visuals in sync with form state when status updates.
	if a.burstButton != nil || a.continuousButton != nil || a.gravityButton != nil {
		a.syncToggleButtons()
	}
	if a.ui != nil && a.ui.Container != nil {
		a.ui.Container.RequestRelayout()
	}
}

func (a *App) revalidate(updatePreview bool) {
	validated, err := a.validateForm()
	if err != nil {
		a.validation = err.Error()
		a.syncStatus()
		return
	}
	a.validation = ""
	if updatePreview {
		if err := a.preview.SetSpec(validated.Spec); err != nil {
			a.validation = err.Error()
			a.syncStatus()
			return
		}
		a.previewStateText = "Playing"
	}
	a.syncStatus()
}

func (a *App) validateForm() (validatedEmitter, error) {
	name := strings.TrimSpace(a.form.Name)
	if name == "" {
		return validatedEmitter{}, fmt.Errorf("name is required")
	}
	totalParticles, err := parsePositiveInt(a.form.TotalParticles, "total_particles")
	if err != nil {
		return validatedEmitter{}, err
	}
	lifetime, err := parsePositiveInt(a.form.Lifetime, "lifetime")
	if err != nil {
		return validatedEmitter{}, err
	}
	colorValue := strings.TrimSpace(a.form.Color)
	if _, err := parseNRGBA(colorValue); err != nil {
		return validatedEmitter{}, err
	}
	spec := prefabs.ParticleEmitterComponentSpec{
		TotalParticles: totalParticles,
		Lifetime:       lifetime,
		Burst:          a.form.Burst,
		Continuous:     a.form.Continuous,
		HasGravity:     a.form.HasGravity,
		Image:          strings.TrimSpace(a.form.Image),
		Color:          colorValue,
	}
	// parse scale values (default to 1.0)
	sx := 1.0
	sy := 1.0
	if strings.TrimSpace(a.form.ScaleX) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(a.form.ScaleX), 64)
		if err != nil {
			return validatedEmitter{}, fmt.Errorf("scale_x must be a number")
		}
		sx = v
	}
	if strings.TrimSpace(a.form.ScaleY) != "" {
		v, err := strconv.ParseFloat(strings.TrimSpace(a.form.ScaleY), 64)
		if err != nil {
			return validatedEmitter{}, fmt.Errorf("scale_y must be a number")
		}
		sy = v
	}
	spec.Scale = struct {
		X float64 `yaml:"x"`
		Y float64 `yaml:"y"`
	}{X: sx, Y: sy}
	return validatedEmitter{Name: name, Spec: spec}, nil
}

func (a *App) saveCurrentEmitter() {
	validated, err := a.validateForm()
	if err != nil {
		a.validation = err.Error()
		a.status = "Save failed"
		a.syncStatus()
		return
	}
	if err := a.preview.SetSpec(validated.Spec); err != nil {
		a.validation = err.Error()
		a.status = "Save failed"
		a.syncStatus()
		return
	}
	path, err := saveEmitterPrefab(a.workspaceRoot, a.prefabDir, validated.Name, buildEmitterPrefab(validated.Name, validated.Spec))
	if err != nil {
		a.validation = err.Error()
		a.status = "Save failed"
		a.syncStatus()
		return
	}
	a.validation = ""
	a.status = fmt.Sprintf("Saved %s", filepath.ToSlash(path))
	a.previewStateText = "Playing"
	a.dirty = false
	a.syncStatus()
}

func buildEmitterPrefab(name string, emitter prefabs.ParticleEmitterComponentSpec) prefabs.EntityBuildSpec {
	return prefabs.EntityBuildSpec{
		Name: strings.TrimSpace(name),
		Components: map[string]any{
			"transform": prefabs.TransformComponentSpec{
				X:        0,
				Y:        0,
				ScaleX:   1,
				ScaleY:   1,
				Rotation: 0,
			},
			"particle_emitter": emitter,
		},
	}
}

func saveEmitterPrefab(workspaceRoot, prefabDir, target string, spec prefabs.EntityBuildSpec) (string, error) {
	normalized, err := editorio.NormalizePrefabTarget(target)
	if err != nil {
		return "", err
	}
	if spec.Components == nil {
		spec.Components = map[string]any{}
	}
	data, err := yaml.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshal prefab %q: %w", normalized, err)
	}
	path := filepath.Join(workspaceRoot, prefabDir, normalized)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create prefab dir for %q: %w", normalized, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write prefab %q: %w", normalized, err)
	}
	return path, nil
}

func parsePositiveInt(value string, fieldName string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", fieldName)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", fieldName)
	}
	return parsed, nil
}

func parseNRGBA(value string) (color.NRGBA, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, nil
	}
	parsed, err := parseHexColor(trimmed)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("color: %w", err)
	}
	return parsed, nil
}

func parseHexColor(v string) (color.NRGBA, error) {
	s := strings.TrimPrefix(strings.TrimSpace(v), "#")
	if len(s) != 6 && len(s) != 8 {
		return color.NRGBA{}, fmt.Errorf("invalid color format: %q", v)
	}
	parse := func(start int) (uint8, error) {
		n, err := strconv.ParseUint(s[start:start+2], 16, 8)
		return uint8(n), err
	}
	r, err := parse(0)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("parse red component: %w", err)
	}
	g, err := parse(2)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("parse green component: %w", err)
	}
	b, err := parse(4)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("parse blue component: %w", err)
	}
	a := uint8(255)
	if len(s) == 8 {
		a, err = parse(6)
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("parse alpha component: %w", err)
		}
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

func isSaveShortcutPressed() bool {
	ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	return ctrl && inpututil.IsKeyJustPressed(ebiten.KeyS)
}

func newLabeledInput(parent *widget.Container, theme *editorcomponents.Theme, label string, placeholder string, onChanged func(string)) *widget.TextInput {
	parent.AddChild(newTextLabel(theme, label, &theme.Face, theme.TextColor))
	input := widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.Placeholder(placeholder),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChanged != nil {
				onChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChanged != nil {
				onChanged(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	parent.AddChild(input)
	return input
}

// newInlineTwoInputs adds a label and two text inputs on the same row.
func newInlineTwoInputs(parent *widget.Container, theme *editorcomponents.Theme, label string, placeholderA, placeholderB string, onChangedA func(string), onChangedB func(string)) (*widget.TextInput, *widget.TextInput) {
	parent.AddChild(newTextLabel(theme, label, &theme.Face, theme.TextColor))
	row := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(8),
		)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	inputA := widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.Placeholder(placeholderA),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChangedA != nil {
				onChangedA(args.InputText)
			}
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChangedA != nil {
				onChangedA(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	inputB := widget.NewTextInput(
		widget.TextInputOpts.Image(theme.InputImage),
		widget.TextInputOpts.Face(&theme.Face),
		widget.TextInputOpts.Color(theme.InputColor),
		widget.TextInputOpts.Padding(widget.NewInsetsSimple(6)),
		widget.TextInputOpts.Placeholder(placeholderB),
		widget.TextInputOpts.ChangedHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChangedB != nil {
				onChangedB(args.InputText)
			}
		}),
		widget.TextInputOpts.SubmitHandler(func(args *widget.TextInputChangedEventArgs) {
			if onChangedB != nil {
				onChangedB(args.InputText)
			}
		}),
		widget.TextInputOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
	row.AddChild(inputA)
	row.AddChild(inputB)
	parent.AddChild(row)
	return inputA, inputB
}

func newActionButton(theme *editorcomponents.Theme, label string, onClick func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonImage),
		widget.ButtonOpts.Text(label, &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if onClick != nil {
				onClick()
			}
		}),
	)
}

func newToggleButton(theme *editorcomponents.Theme, onClick func()) *widget.Button {
	return widget.NewButton(
		widget.ButtonOpts.Image(theme.ButtonImage),
		widget.ButtonOpts.Text("", &theme.Face, theme.ButtonText),
		widget.ButtonOpts.TextPadding(theme.ButtonPadding),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) {
			if onClick != nil {
				onClick()
			}
		}),
	)
}

func setToggleState(button *widget.Button, theme *editorcomponents.Theme, active bool, label string) {
	if button == nil {
		return
	}
	if active {
		button.SetImage(theme.ActiveButtonImage)
		button.SetText(label + ": On")
		return
	}
	button.SetImage(theme.ButtonImage)
	button.SetText(label + ": Off")
}

func newTextLabel(theme *editorcomponents.Theme, label string, face *textv2.Face, textColor color.Color) *widget.Text {
	return widget.NewText(
		widget.TextOpts.Text(label, face, textColor),
		widget.TextOpts.WidgetOpts(widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true})),
	)
}

func drawText(screen *ebiten.Image, value string, face textv2.Face, x, y float64, clr color.Color) {
	var options textv2.DrawOptions
	options.GeoM.Translate(x, y)
	options.ColorScale.ScaleWithColor(clr)
	textv2.Draw(screen, value, face, &options)
}
