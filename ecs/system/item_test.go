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
	"github.com/milk9111/sidescroller/ecs/entity"
	"golang.org/x/image/font/gofont/goregular"
)

func TestItemSystemShowsOverlayAndCollectsPickupOnClose(t *testing.T) {
	w := ecs.NewWorld()

	_ = ecs.CreateEntity(w)

	itemEntity := ecs.CreateEntity(w)
	itemImage := ebiten.NewImage(24, 24)
	if err := ecs.Add(w, itemEntity, component.ItemComponent.Kind(), &component.Item{Description: "test item", Image: itemImage}); err != nil {
		t.Fatalf("add item: %v", err)
	}
	if err := ecs.Add(w, itemEntity, component.PickupComponent.Kind(), &component.Pickup{Kind: "gear"}); err != nil {
		t.Fatalf("add pickup: %v", err)
	}
	if err := ecs.Add(w, itemEntity, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(16, 16)}); err != nil {
		t.Fatalf("add item sprite: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.ItemPopupComponent.Kind(), &component.ItemPopup{TargetItemEntity: uint64(itemEntity)}); err != nil {
		t.Fatalf("add popup: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false, Image: ebiten.NewImage(8, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	_, itemUI, itemState, dialogueInput := addTestItemUI(t, w)

	system := NewItemSystem()
	system.Update(w)

	if !itemState.Active {
		t.Fatal("expected item overlay to become active")
	}
	if itemUI.Text.Label != "test item" {
		t.Fatalf("expected item text to be shown, got %q", itemUI.Text.Label)
	}
	if itemUI.Image.Image != itemImage {
		t.Fatal("expected item image to be assigned to the item UI")
	}
	if itemUI.Overlay.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected item overlay to be visible")
	}

	dialogueInput.Pressed = true
	system.Update(w)

	if itemState.Active {
		t.Fatal("expected item overlay to close on second interaction press")
	}
	if itemUI.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected item overlay to be hidden after close")
	}
	if got := currentPlayerGearCount(w); got != 1 {
		t.Fatalf("expected gear count to increment to 1, got %d", got)
	}
	if _, ok := ecs.Get(w, itemEntity, component.ItemComponent.Kind()); ok {
		t.Fatal("expected item component to be removed after collection")
	}
	if _, ok := ecs.Get(w, itemEntity, component.PickupComponent.Kind()); ok {
		t.Fatal("expected pickup component to be removed after collection")
	}
	if _, ok := ecs.Get(w, itemEntity, component.TTLComponent.Kind()); !ok {
		t.Fatal("expected collected item to receive a ttl")
	}
}

func TestItemSystemEmitsOnItemPickedUpSignalForScriptedItems(t *testing.T) {
	w := ecs.NewWorld()
	scheduler := ecs.NewScheduler(NewItemSystem(), NewScriptSystem())

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "player"}); err != nil {
		t.Fatalf("add player game id: %v", err)
	}

	itemEntity, err := entity.BuildEntity(w, "pickup_item.yaml")
	if err != nil {
		t.Fatalf("build pickup item prefab: %v", err)
	}
	item, ok := ecs.Get(w, itemEntity, component.ItemComponent.Kind())
	if !ok || item == nil {
		t.Fatal("expected item component on pickup item prefab")
	}
	item.Description = "Gear"
	if err := ecs.Add(w, itemEntity, component.ItemComponent.Kind(), item); err != nil {
		t.Fatalf("update item component: %v", err)
	}
	if err := ecs.Add(w, itemEntity, component.ScriptComponent.Kind(), &component.Script{Path: "items/gear.tengo"}); err != nil {
		t.Fatalf("add gear script: %v", err)
	}

	popup := ecs.CreateEntity(w)
	if err := ecs.Add(w, popup, component.ItemPopupComponent.Kind(), &component.ItemPopup{TargetItemEntity: uint64(itemEntity)}); err != nil {
		t.Fatalf("add popup: %v", err)
	}
	if err := ecs.Add(w, popup, component.SpriteComponent.Kind(), &component.Sprite{Disabled: false, Image: ebiten.NewImage(8, 8)}); err != nil {
		t.Fatalf("add popup sprite: %v", err)
	}

	_, itemUI, itemState, dialogueInput := addTestItemUI(t, w)

	scheduler.Update(w)

	if !itemState.Active {
		t.Fatal("expected item overlay to become active")
	}
	if itemUI.Overlay.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected item overlay to be visible")
	}

	dialogueInput.Pressed = true
	scheduler.Update(w)

	if itemState.Active {
		t.Fatal("expected item overlay to close after confirming pickup")
	}
	if got := currentPlayerGearCount(w); got != 1 {
		t.Fatalf("expected scripted item pickup to increment gear count to 1, got %d", got)
	}
	queue, ok := ecs.Get(w, itemEntity, component.ScriptSignalQueueComponent.Kind())
	if !ok || queue == nil || len(queue.Events) != 0 {
		t.Fatalf("expected item signal queue to be drained after script update, got %+v", queue)
	}
	if _, ok := ecs.Get(w, itemEntity, component.TTLComponent.Kind()); !ok {
		t.Fatal("expected scripted item pickup to receive a ttl")
	}
	if itemUI.Overlay.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected item overlay to be hidden after close")
	}
}

func addTestItemUI(t *testing.T, w *ecs.World) (ecs.Entity, *component.ItemUI, *component.ItemState, *component.DialogueInput) {
	t.Helper()

	fontSource, err := textv2.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatalf("load test font: %v", err)
	}
	bodyFace := textv2.Face(&textv2.GoTextFace{Source: fontSource, Size: 16})

	overlay := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	overlay.GetWidget().Visibility = widget.Visibility_Hide
	graphic := widget.NewGraphic(widget.GraphicOpts.Image(ebiten.NewImage(1, 1)))
	graphic.GetWidget().Visibility = widget.Visibility_Hide
	text := widget.NewText(
		widget.TextOpts.Text("", &bodyFace, color.NRGBA{R: 255, G: 255, B: 255, A: 255}),
	)
	overlay.AddChild(graphic)
	overlay.AddChild(text)

	ent := ecs.CreateEntity(w)
	ui := &component.ItemUI{Root: overlay, Overlay: overlay, Image: graphic, Text: text}
	state := &component.ItemState{}
	dialogueInput := &component.DialogueInput{Pressed: true}
	if err := ecs.Add(w, ent, component.ItemUIComponent.Kind(), ui); err != nil {
		t.Fatalf("add item ui: %v", err)
	}
	if err := ecs.Add(w, ent, component.ItemStateComponent.Kind(), state); err != nil {
		t.Fatalf("add item state: %v", err)
	}
	if err := ecs.Add(w, ent, component.DialogueInputComponent.Kind(), dialogueInput); err != nil {
		t.Fatalf("add interaction input: %v", err)
	}

	return ent, ui, state, dialogueInput
}
