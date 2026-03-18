package components

import (
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
)

func TestPanelTextHelpersClampWidth(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	valueText := newValueText(theme)
	if valueText.MaxWidth != scrollableListMaxWidth {
		t.Fatalf("expected value text max width %v, got %v", scrollableListMaxWidth, valueText.MaxWidth)
	}

	layoutData, ok := valueText.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatalf("expected RowLayoutData for value text, got %T", valueText.GetWidget().LayoutData)
	}
	if !layoutData.Stretch || layoutData.MaxWidth != scrollableListMaxWidth {
		t.Fatalf("expected stretched value text capped at %v, got %+v", scrollableListMaxWidth, layoutData)
	}
}

func TestPanelTextInputsClampWidth(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	input := newEditorTextInput(theme, nil)
	layoutData, ok := input.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatalf("expected RowLayoutData for text input, got %T", input.GetWidget().LayoutData)
	}
	if !layoutData.Stretch || layoutData.MaxWidth != scrollableListMaxWidth {
		t.Fatalf("expected stretched input capped at %v, got %+v", scrollableListMaxWidth, layoutData)
	}
}

func TestTransitionPanelDraftStateUsesLiveInputValues(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	panel := NewTransitionPanel(theme, LayerCallbacks{})
	panel.currentState = TransitionEditorState{Selected: true, ID: "t1", ToLevel: "zone_a", LinkedID: "right", EnterDir: "down"}
	panel.IDInput.SetText("t1")
	panel.ToLevelInput.SetText("zone_b")
	panel.LinkedInput.SetText("upper_right")

	state, ok := panel.DraftState()
	if !ok {
		t.Fatal("expected draft state to be available for selected transition")
	}
	if state.LinkedID != "upper_right" {
		t.Fatalf("expected live linked_id upper_right, got %q", state.LinkedID)
	}
	if state.ToLevel != "zone_b" {
		t.Fatalf("expected live to_level zone_b, got %q", state.ToLevel)
	}
	if state.EnterDir != "down" {
		t.Fatalf("expected enter_dir down, got %q", state.EnterDir)
	}
}

func TestTransitionPanelSyncPreservesLocalDraftWhenWorldStateIsStale(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	panel := NewTransitionPanel(theme, LayerCallbacks{})
	panel.Sync(true, nil, 7, TransitionEditorState{Selected: true, ID: "t1", ToLevel: "zone_a", LinkedID: "right", EnterDir: "down"})
	panel.LinkedInput.SetText("upper_right")
	panel.currentState.LinkedID = "upper_right"
	panel.currentState.Selected = true
	panel.draftDirty = true

	panel.Sync(true, nil, 7, TransitionEditorState{Selected: true, ID: "t1", ToLevel: "zone_a", LinkedID: "right", EnterDir: "down"})
	state, ok := panel.DraftState()
	if !ok {
		t.Fatal("expected draft state after sync")
	}
	if state.LinkedID != "upper_right" {
		t.Fatalf("expected local draft linked_id upper_right, got %q", state.LinkedID)
	}
	if !panel.draftDirty {
		t.Fatal("expected draftDirty to remain set while world state is stale")
	}
}

func TestNewAssetPanelFiltersNonTileAssets(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	panel := NewAssetPanel(theme, []editorio.AssetInfo{
		{Name: "grass_tile.png", Relative: "assets/grass_tile.png", DiskPath: "/tmp/grass_tile.png"},
		{Name: "player.png", Relative: "assets/player.png", DiskPath: "/tmp/player.png"},
		{Name: "lab.png", Relative: "tiles/lab_floor.png", DiskPath: "/tmp/lab.png"},
	}, nil, nil, nil)

	if len(panel.entries) != 2 {
		t.Fatalf("expected 2 tile assets in the list, got %d", len(panel.entries))
	}
	first, _ := panel.entries[0].(editorio.AssetInfo)
	second, _ := panel.entries[1].(editorio.AssetInfo)
	if first.Name != "grass_tile.png" {
		t.Fatalf("expected first tile asset to be grass_tile.png, got %q", first.Name)
	}
	if second.Name != "lab.png" {
		t.Fatalf("expected second tile asset to be lab.png, got %q", second.Name)
	}
}

func TestAssetPanelSyncDisablesAssetInteractionsWhileInspectorActive(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	panel := NewAssetPanel(theme, []editorio.AssetInfo{{Name: "grass_tile.png", Relative: "assets/grass_tile.png", DiskPath: "/tmp/grass_tile.png"}}, nil, nil, nil)
	panel.Tileset.buttons = []*widget.Button{widget.NewButton()}
	panel.Tileset.enabled = true
	panel.Tileset.applyEnabledState()

	panel.Sync(model.TileSelection{Path: "grass_tile.png", Index: 0}, false, InspectorState{Active: true, DocumentText: "transform:\n  x: 1"})

	if panel.interactive {
		t.Fatal("expected asset panel interactions to be disabled while inspector is active")
	}
	if !panel.list.GetWidget().Disabled {
		t.Fatal("expected asset list to be disabled while inspector is active")
	}
	if panel.searchList.List.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected elevated asset list to be hidden while inspector is active")
	}
	if panel.searchList.Input.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected asset search input to be hidden while inspector is active")
	}
	if panel.Tileset.enabled {
		t.Fatal("expected tileset picker to be disabled while inspector is active")
	}
	if panel.Tileset.Root.GetWidget().Visibility != widget.Visibility_Hide {
		t.Fatal("expected tileset picker to be hidden while inspector is active")
	}
	if !panel.Tileset.buttons[0].GetWidget().Disabled {
		t.Fatal("expected tileset buttons to be disabled while inspector is active")
	}

	panel.Sync(model.TileSelection{Path: "grass_tile.png", Index: 0}, false, InspectorState{})

	if !panel.interactive {
		t.Fatal("expected asset panel interactions to be restored when inspector closes")
	}
	if panel.list.GetWidget().Disabled {
		t.Fatal("expected asset list to be re-enabled when inspector closes")
	}
	if !panel.Tileset.enabled {
		t.Fatal("expected tileset picker to be re-enabled when inspector closes")
	}
	if panel.SearchInput.GetWidget().Disabled {
		t.Fatal("expected asset search input to be re-enabled when inspector closes")
	}
	if panel.searchList.List.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected asset list to be visible when inspector closes")
	}
	if panel.searchList.Input.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected asset search input to be visible when inspector closes")
	}
	if panel.Tileset.Root.GetWidget().Visibility != widget.Visibility_Show {
		t.Fatal("expected tileset picker to be visible when inspector closes")
	}
	if !panel.searchList.List.GetWidget().ElevateLayer {
		t.Fatal("expected asset list to elevate to its own input layer")
	}
}

func TestSearchableListFiltersEntries(t *testing.T) {
	entries := filterSearchableEntries([]any{"alpha", "beta", "gamma"}, "et", func(entry any) string {
		value, _ := entry.(string)
		return value
	})

	if len(entries) != 1 {
		t.Fatalf("expected 1 filtered entry, got %d", len(entries))
	}
	if got, _ := entries[0].(string); got != "beta" {
		t.Fatalf("expected filtered entry beta, got %v", entries[0])
	}
}

func TestScrollableListElevatesInputLayer(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	list := newScrollableList(theme, nil, func(entry any) string { return "" }, nil)
	if !list.GetWidget().ElevateLayer {
		t.Fatal("expected scrollable list to elevate to its own input layer")
	}
}

func TestPrefabPanelUsesTallerListHeight(t *testing.T) {
	theme, err := NewTheme()
	if err != nil {
		t.Fatalf("NewTheme() error = %v", err)
	}

	panel := NewPrefabPanel(theme, nil)
	if panel.List.GetWidget().MinHeight != 220 {
		t.Fatalf("expected prefab list min height 220, got %d", panel.List.GetWidget().MinHeight)
	}
	layoutData, ok := panel.List.GetWidget().LayoutData.(widget.RowLayoutData)
	if !ok {
		t.Fatalf("expected RowLayoutData for prefab list, got %T", panel.List.GetWidget().LayoutData)
	}
	if layoutData.MaxHeight != 220 {
		t.Fatalf("expected prefab list max height 220, got %d", layoutData.MaxHeight)
	}
}
