package system

import (
	"bytes"
	"image/color"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"golang.org/x/image/font/gofont/goregular"
)

func TestPlayerHealthBarSystemUpdatesHUDWidgets(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.HealthComponent.Kind(), &component.Health{Initial: 3, Current: 3}); err != nil {
		t.Fatalf("add health: %v", err)
	}
	abilitiesEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, abilitiesEntity, component.AbilitiesComponent.Kind(), &component.Abilities{Heal: true}); err != nil {
		t.Fatalf("add abilities: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{HealUses: 1}); err != nil {
		t.Fatalf("add player state machine: %v", err)
	}

	bar := ecs.CreateEntity(w)
	if err := ecs.Add(w, bar, component.PlayerHealthBarComponent.Kind(), &component.PlayerHealthBar{MaxHearts: 3, LastHealth: 3, LastGearCount: 0, LastHealUses: 0, LastCanHeal: false}); err != nil {
		t.Fatalf("add health bar: %v", err)
	}

	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatalf("load font source: %v", err)
	}
	face := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 20})
	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	gearText := widget.NewText(widget.TextOpts.Text("0", &face, color.White))
	heartFull := ebiten.NewImage(8, 8)
	heartEmpty := ebiten.NewImage(8, 8)
	flaskFull := ebiten.NewImage(8, 8)
	flaskEmpty := ebiten.NewImage(8, 8)
	hearts := []*widget.Graphic{
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
	}
	flasks := []*widget.Graphic{
		widget.NewGraphic(widget.GraphicOpts.Image(flaskFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(flaskFull)),
	}
	if err := ecs.Add(w, bar, component.PlayerHUDUIComponent.Kind(), &component.PlayerHUDUI{Root: root, Hearts: hearts, HeartFullImage: heartFull, HeartEmptyImage: heartEmpty, GearText: gearText, Flasks: flasks, FlaskFullImage: flaskFull, FlaskEmptyImage: flaskEmpty}); err != nil {
		t.Fatalf("add player hud ui: %v", err)
	}

	gears := ensurePlayerGearCount(w)
	if gears == nil {
		t.Fatal("expected gear count entity")
	}
	gears.Count = 5

	NewPlayerHealthBarSystem().Update(w)

	barComp, ok := ecs.Get(w, bar, component.PlayerHealthBarComponent.Kind())
	if !ok || barComp == nil {
		t.Fatal("expected health bar component")
	}
	if barComp.LastGearCount != 5 {
		t.Fatalf("expected cached gear count 5, got %d", barComp.LastGearCount)
	}
	if barComp.LastHealth != 3 {
		t.Fatalf("expected cached health 3, got %d", barComp.LastHealth)
	}
	if barComp.LastHealUses != 1 {
		t.Fatalf("expected cached heal uses 1, got %d", barComp.LastHealUses)
	}
	if !barComp.LastCanHeal {
		t.Fatal("expected healing ability cache to be enabled")
	}

	hud, ok := ecs.Get(w, bar, component.PlayerHUDUIComponent.Kind())
	if !ok || hud == nil {
		t.Fatal("expected player hud ui component")
	}
	if hud.GearText == nil {
		t.Fatal("expected gear text widget")
	}
	if hud.GearText.Label != "5" {
		t.Fatalf("expected gear label 5, got %q", hud.GearText.Label)
	}
	for index, heart := range hud.Hearts {
		if heart == nil {
			t.Fatalf("expected heart widget at %d", index)
		}
		if heart.Image != heartFull {
			t.Fatalf("expected heart %d to use full image", index)
		}
	}
	if len(hud.Flasks) != 2 {
		t.Fatalf("expected 2 flask widgets, got %d", len(hud.Flasks))
	}
	if hud.Flasks[0].Image != flaskEmpty {
		t.Fatal("expected first flask to use empty image after one heal use")
	}
	if hud.Flasks[1].Image != flaskFull {
		t.Fatal("expected second flask to remain full")
	}
	if hud.Flasks[0].GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected first flask to be visible when healing is unlocked")
	}
	if hud.Flasks[1].GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected second flask to be visible when healing is unlocked")
	}
}

func TestPlayerHealthBarSystemHidesFlasksUntilHealingUnlocked(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.HealthComponent.Kind(), &component.Health{Initial: 3, Current: 3}); err != nil {
		t.Fatalf("add health: %v", err)
	}
	abilitiesEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, abilitiesEntity, component.AbilitiesComponent.Kind(), &component.Abilities{Heal: false}); err != nil {
		t.Fatalf("add abilities: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{HealUses: 1}); err != nil {
		t.Fatalf("add player state machine: %v", err)
	}

	bar := ecs.CreateEntity(w)
	if err := ecs.Add(w, bar, component.PlayerHealthBarComponent.Kind(), &component.PlayerHealthBar{MaxHearts: 3, LastHealth: 3, LastGearCount: 0, LastHealUses: 0, LastCanHeal: true}); err != nil {
		t.Fatalf("add health bar: %v", err)
	}

	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatalf("load font source: %v", err)
	}
	face := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 20})
	root := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	gearText := widget.NewText(widget.TextOpts.Text("0", &face, color.White))
	heartFull := ebiten.NewImage(8, 8)
	heartEmpty := ebiten.NewImage(8, 8)
	flaskFull := ebiten.NewImage(8, 8)
	flaskEmpty := ebiten.NewImage(8, 8)
	hearts := []*widget.Graphic{
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
	}
	flasks := []*widget.Graphic{
		widget.NewGraphic(widget.GraphicOpts.Image(flaskFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(flaskFull)),
	}
	if err := ecs.Add(w, bar, component.PlayerHUDUIComponent.Kind(), &component.PlayerHUDUI{Root: root, Hearts: hearts, HeartFullImage: heartFull, HeartEmptyImage: heartEmpty, GearText: gearText, Flasks: flasks, FlaskFullImage: flaskFull, FlaskEmptyImage: flaskEmpty}); err != nil {
		t.Fatalf("add player hud ui: %v", err)
	}

	NewPlayerHealthBarSystem().Update(w)

	hud, ok := ecs.Get(w, bar, component.PlayerHUDUIComponent.Kind())
	if !ok || hud == nil {
		t.Fatal("expected player hud ui component")
	}
	for index, flask := range hud.Flasks {
		if flask == nil {
			t.Fatalf("expected flask widget at %d", index)
		}
		if flask.GetWidget().Visibility != widget.Visibility_Hide {
			t.Fatalf("expected flask %d to stay hidden before healing unlock", index)
		}
	}

	abilities, _ := ecs.Get(w, abilitiesEntity, component.AbilitiesComponent.Kind())
	abilities.Heal = true

	NewPlayerHealthBarSystem().Update(w)

	for index, flask := range hud.Flasks {
		if flask.GetWidget().Visibility != widget.Visibility_Show {
			t.Fatalf("expected flask %d to become visible after healing unlock", index)
		}
	}
	if hud.Flasks[0].Image != flaskEmpty {
		t.Fatal("expected first flask to use empty image after unlock with one spent heal")
	}
	if hud.Flasks[1].Image != flaskFull {
		t.Fatal("expected second flask to use full image after unlock")
	}
}
