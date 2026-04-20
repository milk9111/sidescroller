package entity

import (
	"fmt"
	"strings"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const TutorialDefaultFrames = 30 * 60

func ShowTutorial(w *ecs.World, message string, frames int) error {
	if w == nil {
		return fmt.Errorf("tutorial ui: world is nil")
	}

	message = strings.TrimSpace(message)
	if message == "" {
		return fmt.Errorf("tutorial ui: message cannot be empty")
	}

	ent, state, ui, ok := tutorialUIState(w)
	if !ok {
		return fmt.Errorf("tutorial ui: not found")
	}

	if ui.Text != nil {
		ui.Text.Label = message
	}
	setTutorialWidgetVisible(ui.Overlay, true)
	requestTutorialUIRelayout(ui)

	state.Active = true
	if frames > 0 {
		state.RemainingFrames = frames
	} else {
		state.RemainingFrames = -1
	}
	_ = ecs.Add(w, ent, component.TutorialStateComponent.Kind(), state)
	return nil
}

func HideTutorial(w *ecs.World) error {
	if w == nil {
		return fmt.Errorf("tutorial ui: world is nil")
	}

	_, state, ui, ok := tutorialUIState(w)
	if !ok {
		return nil
	}

	if ui.Text != nil {
		ui.Text.Label = ""
	}
	setTutorialWidgetVisible(ui.Overlay, false)
	requestTutorialUIRelayout(ui)

	state.Active = false
	state.RemainingFrames = 0
	return nil
}

func tutorialUIState(w *ecs.World) (ecs.Entity, *component.TutorialState, *component.TutorialUI, bool) {
	if w == nil {
		return 0, nil, nil, false
	}

	ent, ok := ecs.First(w, component.TutorialStateComponent.Kind())
	if !ok {
		return 0, nil, nil, false
	}

	state, ok := ecs.Get(w, ent, component.TutorialStateComponent.Kind())
	if !ok || state == nil {
		return 0, nil, nil, false
	}

	ui, ok := ecs.Get(w, ent, component.TutorialUIComponent.Kind())
	if !ok || ui == nil {
		return 0, nil, nil, false
	}

	return ent, state, ui, true
}

func requestTutorialUIRelayout(ui *component.TutorialUI) {
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

func setTutorialWidgetVisible(node widget.PreferredSizeLocateableWidget, visible bool) {
	if node == nil || node.GetWidget() == nil {
		return
	}
	if visible {
		node.GetWidget().Visibility = widget.Visibility_Show
		return
	}
	node.GetWidget().Visibility = widget.Visibility_Hide
}
