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

	bar := ecs.CreateEntity(w)
	if err := ecs.Add(w, bar, component.PlayerHealthBarComponent.Kind(), &component.PlayerHealthBar{MaxHearts: 3, LastHealth: 3, LastGearCount: 0}); err != nil {
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
	hearts := []*widget.Graphic{
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
		widget.NewGraphic(widget.GraphicOpts.Image(heartFull)),
	}
	if err := ecs.Add(w, bar, component.PlayerHUDUIComponent.Kind(), &component.PlayerHUDUI{Root: root, Hearts: hearts, HeartFullImage: heartFull, HeartEmptyImage: heartEmpty, GearText: gearText}); err != nil {
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
}
