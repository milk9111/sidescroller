package main

import (
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// ToolBar contains the radio-group state for the floating tool buttons.
type ToolBar struct {
	group   *widget.RadioGroup
	buttons []*widget.Button
}

func (tb *ToolBar) SetTool(t Tool) {
	idx := int(t)
	if tb == nil || tb.group == nil || idx < 0 || idx >= len(tb.buttons) {
		return
	}
	tb.group.SetActive(tb.buttons[idx])
}

// TransitionUI holds the widgets for transition mode + transition properties.
type TransitionUI struct {
	modeBtn *widget.Button
	form    *widget.Container

	// relayoutTarget is the container we ask to recompute layout when the
	// transition form becomes visible/hidden.
	relayoutTarget *widget.Container

	idInput     *widget.TextInput
	levelInput  *widget.TextInput
	linkedInput *widget.TextInput
	// dirGroup and dirButtons implement a radio-group for enter_dir selection
	dirGroup   *widget.RadioGroup
	dirButtons []*widget.Button

	suppress bool
	modeOn   bool
}

// GateUI holds the widgets for gate placement mode.
type GateUI struct {
	modeBtn *widget.Button
	modeOn  bool
}

func (t *TransitionUI) SetMode(enabled bool) {
	if t == nil {
		return
	}
	t.modeOn = enabled
	if t.modeBtn == nil {
		return
	}
	label := "Transitions: Off"
	if enabled {
		label = "Transitions: On"
	}
	if text := t.modeBtn.Text(); text != nil {
		text.Label = label
	}
}

func (t *TransitionUI) SetFormVisible(visible bool) {
	if t == nil || t.form == nil {
		return
	}
	if visible {
		t.form.GetWidget().Visibility = widget.Visibility_Show
	} else {
		t.form.GetWidget().Visibility = widget.Visibility_Hide
	}
	// Visibility changes can affect preferred sizes; request a relayout so the
	// form gets positioned by the parent layout instead of rendering at (0,0).
	if t.relayoutTarget != nil {
		t.relayoutTarget.RequestRelayout()
	}
	// Also mark the form itself dirty so its internal layout recalculates.
	t.form.RequestRelayout()
}

func (t *TransitionUI) SetFields(id, level, linked, dir string) {
	if t == nil {
		return
	}
	t.suppress = true
	if t.idInput != nil {
		t.idInput.SetText(id)
	}
	if t.levelInput != nil {
		t.levelInput.SetText(level)
	}
	if t.linkedInput != nil {
		t.linkedInput.SetText(linked)
	}
	if t.dirGroup != nil && len(t.dirButtons) > 0 {
		// dirButtons correspond to: up, down, left, right
		if dir == "" {
			// clear selection
			t.dirGroup.SetActive(nil)
		} else {
			dirLabels := []string{"up", "down", "left", "right"}
			for i, b := range t.dirButtons {
				if i < len(dirLabels) && dirLabels[i] == dir {
					t.dirGroup.SetActive(b)
					break
				}
			}
		}
	}
	t.suppress = false
}

func (g *GateUI) SetMode(enabled bool) {
	if g == nil {
		return
	}
	g.modeOn = enabled
	if g.modeBtn == nil {
		return
	}
	label := "Gates: Off"
	if enabled {
		label = "Gates: On"
	}
	if text := g.modeBtn.Text(); text != nil {
		text.Label = label
	}
}

// TilesetPanelUI is the composed right-panel widget plus helper closures.
type TilesetPanelUI struct {
	Container                  *widget.Container
	ApplyTileset               func(img *ebiten.Image)
	SetTilesetSelection        func(tileIndex int)
	SetTilesetSelectionEnabled func(enabled bool)
}

// LeftPanelUI is the composed left-panel widget and its stateful helpers.
type LeftPanelUI struct {
	Container     *widget.Container
	LayerPanel    *LayerPanel
	FileNameInput *widget.TextInput
	RenameOverlay *widget.Container
	TransitionUI  *TransitionUI
	GateUI        *GateUI
}
