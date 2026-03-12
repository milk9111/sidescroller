package components

import (
	"image"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
)

func TestInspectorSyncUpdatesDocumentState(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)

	panel.Sync(InspectorState{
		Active:        true,
		EntityLabel:   "player",
		PrefabPath:    "player.yaml",
		DocumentText:  "transform:\n  x: 1\n  y: 2",
		StatusMessage: "Editing effective component YAML",
	})

	if got := panel.Editor.GetText(); got != "transform:\n  x: 1\n  y: 2" {
		t.Fatalf("expected document text to sync, got %q", got)
	}
	if got := panel.StatusText.Label; got != "Editing effective component YAML" {
		t.Fatalf("expected status text to sync, got %q", got)
	}
}

func TestInspectorFocusedInputReturnsNil(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	if got := panel.FocusedInput(); got != nil {
		t.Fatalf("expected no focused text input, got %p", got)
	}
	if panel.AnyInputFocused() {
		t.Fatal("expected AnyInputFocused to report false")
	}
}

func TestInspectorSyncTogglesDocumentVisibility(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	state := InspectorState{StatusMessage: "No editable components found"}

	panel.Sync(state)
	if panel.EmptyText.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected empty state to be visible without a document")
	}
	if panel.Editor.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected document text to be hidden without a document")
	}

	state.Active = true
	state.EntityLabel = "player"
	state.DocumentText = "transform:\n  x: 42"
	panel.Sync(state)

	if panel.EmptyText.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected empty state to be hidden when a document exists")
	}
	if panel.Editor.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected document text to be shown when a document exists")
	}
	if got := panel.Editor.GetText(); got != "transform:\n  x: 42" {
		t.Fatalf("expected document text to update, got %q", got)
	}
}

func TestInspectorAnyInputFocusedTracksEditorFocus(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	panel.Editor.Focus(true)
	if !panel.AnyInputFocused() {
		t.Fatal("expected inspector to report focused input when editor is focused")
	}
}

func TestInspectorSyncClearsDirtyWhenAuthoritativeStateMatchesFocusedEditor(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	panel.Sync(InspectorState{Active: true, EntityLabel: "player", DocumentText: "transform:\n  x: 1"})
	panel.Editor.Focus(true)
	panel.Editor.SetText("transform:\n  x: 4")
	panel.Editor.SetDirty(true)

	panel.Sync(InspectorState{Active: true, EntityLabel: "player", DocumentText: "transform:\n  x: 4", Dirty: false})

	if panel.Editor.IsDirty() {
		t.Fatal("expected authoritative state to clear the dirty flag after save")
	}
	if got := panel.Editor.GetText(); got != "transform:\n  x: 4" {
		t.Fatalf("expected focused editor text to remain in sync, got %q", got)
	}
}

func TestInspectorSyncReloadsFocusedDirtyEditorOnAuthoritativeDocumentChange(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	panel.Sync(InspectorState{Active: true, EntityLabel: "player", DocumentText: "transform:\n  x: 1"})
	panel.Editor.Focus(true)
	panel.Editor.SetText("transform:\n  x: 4")
	panel.Editor.SetDirty(true)

	panel.Sync(InspectorState{Active: true, EntityLabel: "player", DocumentText: "transform:\n  x: 4.0", Dirty: false})

	if panel.Editor.IsDirty() {
		t.Fatal("expected authoritative reload to clear the dirty flag")
	}
	if got := panel.Editor.GetText(); got != "transform:\n  x: 4.0" {
		t.Fatalf("expected focused editor to reload canonical document text, got %q", got)
	}
}

func TestInspectorSyncDoesNotClearDirtyOnlyBecauseTextMatchesCanonical(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	panel.Sync(InspectorState{Active: true, EntityLabel: "player", DocumentText: "transform:\n  x: 4"})
	panel.Editor.Focus(true)
	panel.Editor.SetText("transform:\n  x: 4")
	panel.Editor.SetDirty(true)

	panel.Sync(InspectorState{Active: true, EntityLabel: "player", DocumentText: "transform:\n  x: 4", Dirty: true})

	if !panel.Editor.IsDirty() {
		t.Fatal("expected matching canonical text to remain dirty until a successful apply clears it")
	}
}

func TestInspectorSetAvailableHeightExpandsEditorToFillPanel(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}
	panel := NewInspectorPanel(theme, nil)
	panel.Sync(InspectorState{Active: true, EntityLabel: "player", DocumentText: "transform:\n  x: 1"})
	panel.Root.GetWidget().Rect = image.Rect(0, 0, 248, 720)

	if changed := panel.SetAvailableHeight(720); !changed {
		t.Fatal("expected panel height update to resize the editor")
	}
	if panel.Editor.GetWidget().MinHeight <= textEditorMinHeight {
		t.Fatalf("expected editor minimum height to grow beyond default, got %d", panel.Editor.GetWidget().MinHeight)
	}
	if panel.Root.GetWidget().MinHeight < panel.Editor.GetWidget().MinHeight {
		t.Fatalf("expected inspector root to grow with the editor, got root=%d editor=%d", panel.Root.GetWidget().MinHeight, panel.Editor.GetWidget().MinHeight)
	}
}
