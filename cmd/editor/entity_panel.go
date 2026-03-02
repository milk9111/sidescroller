package main

import "github.com/ebitenui/ebitenui/widget"

// EntityListEntry is a row in the active-entities list.
type EntityListEntry struct {
	Index int
	Type  string
	CellX int
	CellY int
}

// EntityPanel holds the active-entities list widget and helper state.
type EntityPanel struct {
	list           *widget.List
	entries        []any
	lastEntries    []EntityListEntry
	currentSelect  int
	suppressEvents bool
}

func NewEntityPanel() *EntityPanel {
	return &EntityPanel{currentSelect: -1}
}

func (ep *EntityPanel) SetEntries(entries []EntityListEntry) {
	if ep == nil || ep.list == nil {
		return
	}
	if entityEntriesEqual(ep.lastEntries, entries) {
		return
	}
	ep.suppressEvents = true
	anyEntries := make([]any, len(entries))
	for i := range entries {
		anyEntries[i] = entries[i]
	}
	ep.entries = anyEntries
	ep.lastEntries = append(ep.lastEntries[:0], entries...)
	ep.list.SetEntries(anyEntries)
	ep.suppressEvents = false
}

func (ep *EntityPanel) SetSelected(entityIndex int) {
	if ep == nil || ep.list == nil {
		return
	}
	if ep.currentSelect == entityIndex {
		return
	}
	ep.currentSelect = entityIndex
	ep.suppressEvents = true
	if entityIndex < 0 {
		ep.list.SetSelectedEntry(nil)
		ep.suppressEvents = false
		return
	}
	for i := range ep.entries {
		entry, ok := ep.entries[i].(EntityListEntry)
		if !ok {
			continue
		}
		if entry.Index == entityIndex {
			ep.list.SetSelectedEntry(ep.entries[i])
			ep.suppressEvents = false
			return
		}
	}
	ep.list.SetSelectedEntry(nil)
	ep.suppressEvents = false
}

func entityEntriesEqual(a, b []EntityListEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
