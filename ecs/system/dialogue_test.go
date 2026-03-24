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

func TestDialogueSystemStartsAdvancesAndCloses(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.InputComponent.Kind(), &component.Input{}); err != nil {
		t.Fatalf("add input: %v", err)
	}

	speaker := ecs.CreateEntity(w)
	if err := ecs.Add(w, speaker, component.DialogueComponent.Kind(), &component.Dialogue{Lines: []string{"line one", "line two"}}); err != nil {
		t.Fatalf("add dialogue: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.DialoguePopupComponent.Kind(), &component.DialoguePopup{TargetDialogueEntity: uint64(speaker)}); err != nil {
		t.Fatalf("add popup: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false, Image: ebiten.NewImage(8, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	_, dialogueUI, dialogueState, dialogueInput := addTestDialogueUI(t, w)

	system := NewDialogueSystem()
	system.Update(w)

	if !dialogueState.Active {
		t.Fatal("expected dialogue to become active")
	}
	if dialogueState.LineIndex != 0 {
		t.Fatalf("expected first line index, got %d", dialogueState.LineIndex)
	}
	if dialogueUI.Text.Label != "line one" {
		t.Fatalf("expected first line text, got %q", dialogueUI.Text.Label)
	}
	if dialogueUI.Overlay.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected dialogue overlay to be visible")
	}

	popupSprite, ok := ecs.Get(w, popup, component.SpriteComponent.Kind())
	if !ok || popupSprite == nil {
		t.Fatal("expected popup sprite")
	}
	if !popupSprite.Disabled {
		t.Fatal("expected popup sprite to be hidden while dialogue is active")
	}

	dialogueInput.Pressed = true
	system.Update(w)

	if dialogueState.LineIndex != 1 {
		t.Fatalf("expected second line index, got %d", dialogueState.LineIndex)
	}
	if dialogueUI.Text.Label != "line two" {
		t.Fatalf("expected second line text, got %q", dialogueUI.Text.Label)
	}

	dialogueInput.Pressed = true
	system.Update(w)

	if dialogueState.Active {
		t.Fatal("expected dialogue to close after the last line")
	}
	if dialogueUI.Text.Label != "" {
		t.Fatalf("expected dialogue text to clear on close, got %q", dialogueUI.Text.Label)
	}
	if dialogueUI.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected dialogue overlay to be hidden after close")
	}
}

func TestDialogueSystemIgnoresInvalidPopupTarget(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.InputComponent.Kind(), &component.Input{}); err != nil {
		t.Fatalf("add input: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.DialoguePopupComponent.Kind(), &component.DialoguePopup{}); err != nil {
		t.Fatalf("add popup: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false, Image: ebiten.NewImage(8, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	_, dialogueUI, dialogueState, _ := addTestDialogueUI(t, w)

	NewDialogueSystem().Update(w)

	if dialogueState.Active {
		t.Fatal("expected dialogue to remain inactive without a target")
	}
	if dialogueUI.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected overlay to remain hidden")
	}
}

func TestDialogueSystemShowsPortraitWhenPresent(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}

	portrait := ebiten.NewImage(24, 24)
	speaker := ecs.CreateEntity(w)
	if err := ecs.Add(w, speaker, component.DialogueComponent.Kind(), &component.Dialogue{Lines: []string{"portrait line"}, Portrait: portrait}); err != nil {
		t.Fatalf("add dialogue: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.DialoguePopupComponent.Kind(), &component.DialoguePopup{TargetDialogueEntity: uint64(speaker)}); err != nil {
		t.Fatalf("add popup: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false, Image: ebiten.NewImage(8, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	_, dialogueUI, _, _ := addTestDialogueUI(t, w)

	NewDialogueSystem().Update(w)

	if dialogueUI.Portrait == nil {
		t.Fatal("expected portrait widget")
	}
	if dialogueUI.Portrait.Image != portrait {
		t.Fatal("expected portrait image to be assigned to the dialogue UI")
	}
	if dialogueUI.Portrait.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected portrait widget to be visible")
	}
}

func addTestDialogueUI(t *testing.T, w *ecs.World) (ecs.Entity, *component.DialogueUI, *component.DialogueState, *component.DialogueInput) {
	t.Helper()

	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatalf("load test font: %v", err)
	}
	bodyFace := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 16})

	overlay := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	overlay.GetWidget().Visibility = widget.Visibility_Hide
	portrait := widget.NewGraphic(widget.GraphicOpts.Image(ebiten.NewImage(1, 1)))
	portrait.GetWidget().Visibility = widget.Visibility_Hide
	text := widget.NewText(
		widget.TextOpts.Text("", &bodyFace, color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
	)
	overlay.AddChild(portrait)
	overlay.AddChild(text)

	ent := ecs.CreateEntity(w)
	ui := &component.DialogueUI{Root: overlay, Overlay: overlay, Portrait: portrait, Text: text}
	state := &component.DialogueState{}
	dialogueInput := &component.DialogueInput{Pressed: true}
	if err := ecs.Add(w, ent, component.DialogueUIComponent.Kind(), ui); err != nil {
		t.Fatalf("add dialogue ui: %v", err)
	}
	if err := ecs.Add(w, ent, component.DialogueStateComponent.Kind(), state); err != nil {
		t.Fatalf("add dialogue state: %v", err)
	}
	if err := ecs.Add(w, ent, component.DialogueInputComponent.Kind(), dialogueInput); err != nil {
		t.Fatalf("add dialogue input: %v", err)
	}

	return ent, ui, state, dialogueInput
}
