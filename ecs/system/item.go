package system

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type ItemSystem struct{}

func NewItemSystem() *ItemSystem {
	return &ItemSystem{}
}

func IsItemActive(w *ecs.World) bool {
	_, state, _, ok := itemUIState(w)
	return ok && state != nil && state.Active
}

func (s *ItemSystem) Update(w *ecs.World) {
	stateEnt, state, ui, ok := itemUIState(w)
	if !ok || state == nil || ui == nil {
		return
	}

	popupEntity, popup, popupSprite, popupOK := itemPopupState(w)
	pressed, _ := dialogueInputPressed(w)

	if state.Active {
		if popupOK && popupSprite != nil {
			popupSprite.Disabled = true
		}

		itemEntity := ecs.Entity(state.ItemEntity)
		item, sprite, ok := activeItem(w, itemEntity)
		if !ok {
			closeItem(state, ui)
			return
		}

		if sprite != nil {
			sprite.Disabled = true
		}

		showItem(ui, item, sprite)
		if pressed {
			collectItemEntity(w, itemEntity)
			closeItem(state, ui)
			return
		}

		_ = ecs.Add(w, stateEnt, component.ItemStateComponent.Kind(), state)
		return
	}

	hideItemUI(ui)
	if IsDialogueActive(w) || !pressed || !popupOK || popup == nil {
		return
	}

	itemEntity := ecs.Entity(popup.TargetItemEntity)
	item, sprite, ok := activeItem(w, itemEntity)
	if !ok {
		return
	}

	state.Active = true
	state.ItemEntity = uint64(itemEntity)
	showItem(ui, item, sprite)
	if popupSprite != nil {
		popupSprite.Disabled = true
	}
	if sprite != nil {
		sprite.Disabled = true
	}
	_ = ecs.Add(w, stateEnt, component.ItemStateComponent.Kind(), state)
	if popupEntity.Valid() {
		_ = ecs.Add(w, popupEntity, component.ItemPopupComponent.Kind(), popup)
	}
}

func itemUIState(w *ecs.World) (ecs.Entity, *component.ItemState, *component.ItemUI, bool) {
	if w == nil {
		return 0, nil, nil, false
	}

	ent, ok := ecs.First(w, component.ItemStateComponent.Kind())
	if !ok {
		return 0, nil, nil, false
	}

	state, ok := ecs.Get(w, ent, component.ItemStateComponent.Kind())
	if !ok || state == nil {
		return 0, nil, nil, false
	}

	ui, ok := ecs.Get(w, ent, component.ItemUIComponent.Kind())
	if !ok || ui == nil {
		return 0, nil, nil, false
	}

	return ent, state, ui, true
}

func itemPopupState(w *ecs.World) (ecs.Entity, *component.ItemPopup, *component.Sprite, bool) {
	if w == nil {
		return 0, nil, nil, false
	}

	ent, ok := ecs.First(w, component.ItemPopupComponent.Kind())
	if !ok {
		return 0, nil, nil, false
	}

	popup, ok := ecs.Get(w, ent, component.ItemPopupComponent.Kind())
	if !ok || popup == nil {
		return 0, nil, nil, false
	}

	sprite, _ := ecs.Get(w, ent, component.SpriteComponent.Kind())
	return ent, popup, sprite, true
}

func activeItem(w *ecs.World, itemEntity ecs.Entity) (*component.Item, *component.Sprite, bool) {
	if w == nil || !itemEntity.Valid() || !ecs.IsAlive(w, itemEntity) {
		return nil, nil, false
	}

	item, ok := ecs.Get(w, itemEntity, component.ItemComponent.Kind())
	if !ok || item == nil {
		return nil, nil, false
	}

	sprite, _ := ecs.Get(w, itemEntity, component.SpriteComponent.Kind())
	return item, sprite, true
}

func closeItem(state *component.ItemState, ui *component.ItemUI) {
	if state != nil {
		state.Active = false
		state.ItemEntity = 0
	}
	hideItemUI(ui)
}

func hideItemUI(ui *component.ItemUI) {
	if ui == nil {
		return
	}
	if ui.Text != nil {
		ui.Text.Label = ""
	}
	if ui.Image != nil {
		ui.Image.Image = ebiten.NewImage(1, 1)
		setWidgetVisible(ui.Image, false)
	}
	setWidgetVisible(ui.Overlay, false)
	requestItemUIRelayout(ui)
}

func showItem(ui *component.ItemUI, item *component.Item, sprite *component.Sprite) {
	if ui == nil || item == nil {
		return
	}

	if ui.Image != nil {
		image := item.Image
		if image == nil && sprite != nil {
			image = sprite.Image
		}
		if image != nil {
			ui.Image.Image = image
			setWidgetVisible(ui.Image, true)
		} else {
			setWidgetVisible(ui.Image, false)
		}
	}

	if ui.Text != nil {
		ui.Text.Label = strings.TrimSpace(item.Description)
	}

	setWidgetVisible(ui.Overlay, true)
	requestItemUIRelayout(ui)
}

func requestItemUIRelayout(ui *component.ItemUI) {
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
}

func collectItemEntity(w *ecs.World, e ecs.Entity) {
	if w == nil || !e.Valid() || !ecs.IsAlive(w, e) {
		return
	}

	source := itemSignalSource(w, e)
	EmitEntitySignal(w, e, source, "on_item_picked_up")

	if pickup, ok := ecs.Get(w, e, component.PickupComponent.Kind()); ok && pickup != nil {
		collectPickupEntity(w, e, pickup)
		return
	}

	recordLevelEntityState(w, e, component.PersistedLevelEntityStateCollected)
	_ = ecs.Remove(w, e, component.ItemComponent.Kind())
	_ = ecs.Remove(w, e, component.SpriteComponent.Kind())
	_ = ecs.Add(w, e, component.TTLComponent.Kind(), &component.TTL{Frames: 2})
}

func itemSignalSource(w *ecs.World, fallback ecs.Entity) ecs.Entity {
	if w == nil {
		return fallback
	}
	if player, ok := ecs.First(w, component.PlayerTagComponent.Kind()); ok && player.Valid() && ecs.IsAlive(w, player) {
		return player
	}
	return fallback
}
