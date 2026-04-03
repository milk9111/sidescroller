package entity

import (
	"bytes"
	"fmt"
	"image/color"
	"strconv"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	healthBarPaddingX = 12.0
	healthBarPaddingY = 12.0
	heartSpacing      = 4.0
	gearCounterGap    = 8.0
	gearRowSpacing    = 6.0
	healMaxUses       = 2
)

func buildPlayerHUDTextFace() (textv2.Face, error) {
	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return nil, err
	}
	return textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 20}), nil
}

func buildBlackoutImage(src *ebiten.Image) *ebiten.Image {
	if src == nil {
		return nil
	}

	bounds := src.Bounds()
	dst := ebiten.NewImage(bounds.Dx(), bounds.Dy())
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(0, 0, 0, 1)
	dst.DrawImage(src, op)
	return dst
}

func NewPlayerHealthBar(w *ecs.World) (ecs.Entity, error) {
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return 0, nil
	}

	health, ok := ecs.Get(w, player, component.HealthComponent.Kind())
	if !ok || health == nil || health.Initial <= 0 {
		return 0, nil
	}

	heartImage, err := assets.LoadImage("life_heart.png")
	if err != nil {
		return 0, fmt.Errorf("player health bar: load heart sprite: %w", err)
	}
	gearImage, err := assets.LoadImage("gear_icon.png")
	if err != nil {
		return 0, fmt.Errorf("player health bar: load gear sprite: %w", err)
	}
	flaskImage, err := assets.LoadImage("healing_flask_icon.png")
	if err != nil {
		return 0, fmt.Errorf("player health bar: load healing flask sprite: %w", err)
	}
	heartEmptyImage := buildBlackoutImage(heartImage)
	if heartEmptyImage == nil {
		return 0, fmt.Errorf("player health bar: build blackout heart sprite")
	}
	flaskEmptyImage := buildBlackoutImage(flaskImage)
	if flaskEmptyImage == nil {
		return 0, fmt.Errorf("player health bar: build blackout flask sprite")
	}
	textFace, err := buildPlayerHUDTextFace()
	if err != nil {
		return 0, fmt.Errorf("player health bar: load hud font: %w", err)
	}

	uiEnt, ok := ecs.First(w, component.DialogueUIComponent.Kind())
	if !ok {
		return 0, fmt.Errorf("player health bar: ui root not found")
	}
	dialogueUI, ok := ecs.Get(w, uiEnt, component.DialogueUIComponent.Kind())
	if !ok || dialogueUI == nil || dialogueUI.Root == nil {
		return 0, fmt.Errorf("player health bar: ui root unavailable")
	}

	currentHealth := health.Current
	if currentHealth < 0 {
		currentHealth = 0
	}
	if currentHealth > health.Initial {
		currentHealth = health.Initial
	}
	gearCount := 0
	if gearsEntity, ok := ecs.First(w, component.PlayerGearCountComponent.Kind()); ok {
		if gears, ok := ecs.Get(w, gearsEntity, component.PlayerGearCountComponent.Kind()); ok && gears != nil {
			gearCount = gears.Count
		}
	}
	healUses := 0
	canHeal := playerHealAbilityUnlocked(w)
	if stateMachine, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind()); ok && stateMachine != nil {
		healUses = stateMachine.HealUses
	}
	if healUses < 0 {
		healUses = 0
	}
	if healUses > healMaxUses {
		healUses = healMaxUses
	}

	hudRoot := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
	)
	hudRoot.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		Padding:            &widget.Insets{Left: int(healthBarPaddingX), Top: int(healthBarPaddingY)},
	}

	hudContent := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(int(gearCounterGap)),
		)),
	)
	hudContent.GetWidget().LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
	}

	heartsRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(int(heartSpacing)),
		)),
	)
	heartsRow.GetWidget().LayoutData = widget.RowLayoutData{Stretch: false}

	gearRow := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(int(gearRowSpacing)),
		)),
	)
	gearRow.GetWidget().LayoutData = widget.RowLayoutData{Stretch: false}
	flasks := make([]*widget.Graphic, 0, healMaxUses)

	hearts := make([]*widget.Graphic, 0, health.Initial)
	for i := 0; i < health.Initial; i++ {
		img := heartImage
		if i >= currentHealth {
			img = heartEmptyImage
		}
		heartGraphic := widget.NewGraphic(widget.GraphicOpts.Image(img))
		heartGraphic.GetWidget().MinWidth = heartImage.Bounds().Dx()
		heartGraphic.GetWidget().MinHeight = heartImage.Bounds().Dy()
		heartsRow.AddChild(heartGraphic)
		hearts = append(hearts, heartGraphic)
	}

	gearGraphic := widget.NewGraphic(widget.GraphicOpts.Image(gearImage))
	gearGraphic.GetWidget().MinWidth = gearImage.Bounds().Dx()
	gearGraphic.GetWidget().MinHeight = gearImage.Bounds().Dy()
	gearText := widget.NewText(widget.TextOpts.Text(strconv.Itoa(gearCount), &textFace, colorWhite()))

	gearRow.AddChild(gearGraphic)
	gearRow.AddChild(gearText)
	for i := 0; i < healMaxUses; i++ {
		img := flaskImage
		if !canHeal || i < healUses {
			img = flaskEmptyImage
		}
		flaskGraphic := widget.NewGraphic(widget.GraphicOpts.Image(img))
		flaskGraphic.GetWidget().MinWidth = flaskImage.Bounds().Dx()
		flaskGraphic.GetWidget().MinHeight = flaskImage.Bounds().Dy()
		if !canHeal {
			flaskGraphic.GetWidget().Visibility = widget.Visibility_Hide
		}
		gearRow.AddChild(flaskGraphic)
		flasks = append(flasks, flaskGraphic)
	}
	hudContent.AddChild(heartsRow)
	hudContent.AddChild(gearRow)
	hudRoot.AddChild(hudContent)
	hudParent := dialogueUI.Root
	if dialogueUI.HUDLayer != nil {
		hudParent = dialogueUI.HUDLayer
	}
	hudParent.AddChild(hudRoot)
	hudParent.RequestRelayout()

	barEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, barEntity, component.PersistentComponent.Kind(), &component.Persistent{ID: "player_health_bar", KeepOnLevelChange: true, KeepOnReload: false}); err != nil {
		return 0, fmt.Errorf("player health bar: add persistent: %w", err)
	}
	if err := ecs.Add(w, barEntity, component.PlayerHealthBarComponent.Kind(), &component.PlayerHealthBar{MaxHearts: health.Initial, LastHealth: currentHealth, LastGearCount: gearCount, LastHealUses: healUses, LastCanHeal: canHeal}); err != nil {
		return 0, fmt.Errorf("player health bar: add bar component: %w", err)
	}
	if err := ecs.Add(w, barEntity, component.PlayerHUDUIComponent.Kind(), &component.PlayerHUDUI{Root: hudRoot, Hearts: hearts, HeartFullImage: heartImage, HeartEmptyImage: heartEmptyImage, GearText: gearText, Flasks: flasks, FlaskFullImage: flaskImage, FlaskEmptyImage: flaskEmptyImage}); err != nil {
		return 0, fmt.Errorf("player health bar: add hud ui: %w", err)
	}

	return barEntity, nil
}

func colorWhite() color.Color {
	return color.NRGBA{R: 236, G: 240, B: 250, A: 255}
}

func playerHealAbilityUnlocked(w *ecs.World) bool {
	if w == nil {
		return false
	}
	if abilitiesEntity, ok := ecs.First(w, component.AbilitiesComponent.Kind()); ok {
		if abilities, ok := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind()); ok && abilities != nil {
			return abilities.Heal
		}
	}
	return false
}
