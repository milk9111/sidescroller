package system

import (
	"strings"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type DialogueSystem struct{}

func NewDialogueSystem() *DialogueSystem {
	return &DialogueSystem{}
}

func IsDialogueActive(w *ecs.World) bool {
	_, state, _, ok := dialogueUIState(w)
	return ok && state != nil && state.Active
}

func (s *DialogueSystem) Update(w *ecs.World) {
	stateEnt, state, ui, ok := dialogueUIState(w)
	if !ok || state == nil || ui == nil {
		return
	}

	popupEntity, popup, popupSprite, popupOK := dialoguePopupState(w)
	pressed, _ := dialogueInputPressed(w)

	if state.Active {
		if popupOK && popupSprite != nil {
			popupSprite.Disabled = true
		}

		dialogueEntity := ecs.Entity(state.DialogueEntity)
		dialogue, lineCount, ok := activeDialogue(w, dialogueEntity)
		if !ok {
			closeDialogue(state, ui)
			return
		}

		if pressed {
			nextLine := state.LineIndex + 1
			if nextLine >= lineCount {
				closeDialogue(state, ui)
				return
			}
			state.LineIndex = nextLine
		}

		showDialogueLine(ui, dialogue, state.LineIndex)
		_ = ecs.Add(w, stateEnt, component.DialogueStateComponent.Kind(), state)
		return
	}

	hideDialogueUI(ui)
	if !pressed || !popupOK || popup == nil {
		return
	}

	dialogueEntity := ecs.Entity(popup.TargetDialogueEntity)
	dialogue, lineCount, ok := activeDialogue(w, dialogueEntity)
	if !ok || lineCount == 0 {
		return
	}

	state.Active = true
	state.DialogueEntity = uint64(dialogueEntity)
	state.LineIndex = 0
	showDialogueLine(ui, dialogue, state.LineIndex)
	if popupSprite != nil {
		popupSprite.Disabled = true
	}
	_ = ecs.Add(w, stateEnt, component.DialogueStateComponent.Kind(), state)
	if popupEntity.Valid() {
		_ = ecs.Add(w, popupEntity, component.DialoguePopupComponent.Kind(), popup)
	}
}

func dialogueUIState(w *ecs.World) (ecs.Entity, *component.DialogueState, *component.DialogueUI, bool) {
	if w == nil {
		return 0, nil, nil, false
	}

	ent, ok := ecs.First(w, component.DialogueStateComponent.Kind())
	if !ok {
		return 0, nil, nil, false
	}

	state, ok := ecs.Get(w, ent, component.DialogueStateComponent.Kind())
	if !ok || state == nil {
		return 0, nil, nil, false
	}

	ui, ok := ecs.Get(w, ent, component.DialogueUIComponent.Kind())
	if !ok || ui == nil {
		return 0, nil, nil, false
	}

	return ent, state, ui, true
}

func dialoguePopupState(w *ecs.World) (ecs.Entity, *component.DialoguePopup, *component.Sprite, bool) {
	if w == nil {
		return 0, nil, nil, false
	}

	ent, ok := ecs.First(w, component.DialoguePopupComponent.Kind())
	if !ok {
		return 0, nil, nil, false
	}

	popup, ok := ecs.Get(w, ent, component.DialoguePopupComponent.Kind())
	if !ok || popup == nil {
		return 0, nil, nil, false
	}

	sprite, _ := ecs.Get(w, ent, component.SpriteComponent.Kind())
	return ent, popup, sprite, true
}

func dialogueInputPressed(w *ecs.World) (bool, bool) {
	if w == nil {
		return false, false
	}

	ent, ok := ecs.First(w, component.DialogueInputComponent.Kind())
	if !ok {
		return false, false
	}

	input, ok := ecs.Get(w, ent, component.DialogueInputComponent.Kind())
	if !ok || input == nil {
		return false, false
	}

	return input.Pressed, input.UsingGamepad
}

func activeDialogue(w *ecs.World, dialogueEntity ecs.Entity) (*component.Dialogue, int, bool) {
	if w == nil || !dialogueEntity.Valid() || !ecs.IsAlive(w, dialogueEntity) {
		return nil, 0, false
	}

	dialogue, ok := ecs.Get(w, dialogueEntity, component.DialogueComponent.Kind())
	if !ok || dialogue == nil || len(dialogue.Lines) == 0 {
		return nil, 0, false
	}

	return dialogue, len(dialogue.Lines), true
}

func closeDialogue(state *component.DialogueState, ui *component.DialogueUI) {
	if state != nil {
		state.Active = false
		state.DialogueEntity = 0
		state.LineIndex = 0
	}
	hideDialogueUI(ui)
}

func hideDialogueUI(ui *component.DialogueUI) {
	if ui == nil {
		return
	}
	if ui.Text != nil {
		ui.Text.Label = ""
	}
	if ui.PortraitBox != nil {
		setWidgetVisible(ui.PortraitBox, false)
	}
	if ui.Portrait != nil {
		setWidgetVisible(ui.Portrait, false)
	}
	setWidgetVisible(ui.Overlay, false)
	requestDialogueUIRelayout(ui)
}

func showDialogueLine(ui *component.DialogueUI, dialogue *component.Dialogue, lineIndex int) {
	if ui == nil {
		return
	}
	if dialogue == nil || len(dialogue.Lines) == 0 {
		return
	}
	if lineIndex < 0 {
		lineIndex = 0
	}
	if lineIndex >= len(dialogue.Lines) {
		lineIndex = len(dialogue.Lines) - 1
	}

	if ui.Text != nil {
		ui.Text.Label = strings.TrimSpace(dialogue.Lines[lineIndex])
	}
	if ui.Portrait != nil {
		if dialogue.Portrait != nil {
			ui.Portrait.Image = dialogue.Portrait
			if ui.PortraitBox != nil {
				setWidgetVisible(ui.PortraitBox, true)
			}
			setWidgetVisible(ui.Portrait, true)
		} else {
			if ui.PortraitBox != nil {
				setWidgetVisible(ui.PortraitBox, false)
			}
			setWidgetVisible(ui.Portrait, false)
		}
	}

	setWidgetVisible(ui.Overlay, true)
	requestDialogueUIRelayout(ui)
}

func requestDialogueUIRelayout(ui *component.DialogueUI) {
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
	if ui.PortraitBox != nil {
		ui.PortraitBox.RequestRelayout()
	}
}

func setWidgetVisible(node widget.PreferredSizeLocateableWidget, visible bool) {
	if node == nil || node.GetWidget() == nil {
		return
	}

	if visible {
		node.GetWidget().Visibility = widget.Visibility_Show
		return
	}

	node.GetWidget().Visibility = widget.Visibility_Hide
}
