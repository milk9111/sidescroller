package main

import (
	"time"

	"github.com/ebitenui/ebitenui/widget"
)

// LayerEntry is a small value used by the UI list to represent a layer row.
type LayerEntry struct {
	Index int
	Name  string
}

// LayerPanel holds the list widget and small helpers used by the editor UI.
type LayerPanel struct {
	list             *widget.List
	entries          []any
	lastClickTime    time.Time
	lastClickIndex   int
	openRenameDialog func(idx int, current string)

	onNewLayer func()
	onMoveUp   func(idx int)
	onMoveDown func(idx int)
	// suppressEvents, when true, causes the selection handler to avoid
	// interpreting programmatic selections as user clicks/double-clicks.
	suppressEvents bool
}

func NewLayerPanel() *LayerPanel {
	return &LayerPanel{
		lastClickTime:  time.Time{},
		lastClickIndex: -1,
	}
}

func (lp *LayerPanel) SetLayers(names []string) {
	if lp == nil || lp.list == nil {
		return
	}
	// Suppress selection events while we populate the list.
	lp.suppressEvents = true
	entries := make([]any, len(names))
	for i, name := range names {
		entries[i] = LayerEntry{Index: i, Name: name}
	}
	lp.entries = entries
	lp.list.SetEntries(entries)
	// Re-enable event handling after population.
	lp.suppressEvents = false
}

func (lp *LayerPanel) SetSelected(idx int) {
	if lp == nil || lp.list == nil {
		return
	}
	if idx < 0 || idx >= len(lp.entries) {
		return
	}
	// Suppress events around the programmatic selection so it isn't treated
	// as a user click/double-click.
	lp.suppressEvents = true
	lp.lastClickIndex = -1
	lp.lastClickTime = time.Time{}
	lp.list.SetSelectedEntry(lp.entries[idx])
	// Re-enable handling after selection is set.
	lp.suppressEvents = false
}
