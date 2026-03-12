package editorsystem

import (
	"strings"
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestEditorEntitySystemPlacesPrefab(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 2})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{LeftJustPressed: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, CellX: 4, CellY: 6})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy"}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{SelectedPath: "enemy.yaml", SelectedType: "enemy"})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: -1, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})

	NewEditorEntitySystem().Update(w)

	_, entities, _ := entitiesState(w)
	if len(entities.Items) != 1 {
		t.Fatalf("expected one placed entity, got %d", len(entities.Items))
	}
	placed := entities.Items[0]
	if placed.Type != "enemy" {
		t.Fatalf("expected enemy type, got %q", placed.Type)
	}
	if placed.X != 4*TileSize || placed.Y != 6*TileSize {
		t.Fatalf("expected snapped position (%d,%d), got (%d,%d)", 4*TileSize, 6*TileSize, placed.X, placed.Y)
	}
	if got, ok := entityLayerIndex(placed.Props); !ok || got != 2 {
		t.Fatalf("expected entity layer 2, got %v (ok=%t)", got, ok)
	}
	if prefabPath, _ := placed.Props["prefab"].(string); prefabPath != "enemy.yaml" {
		t.Fatalf("expected prefab prop enemy.yaml, got %q", prefabPath)
	}
	_, selection, _ := entitySelectionState(w)
	if selection.SelectedIndex != 0 {
		t.Fatalf("expected selection index 0, got %d", selection.SelectedIndex)
	}
}

func TestEditorEntitySystemDragCapturesSingleSnapshot(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{LeftDown: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, CellX: 2, CellY: 3})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", X: 0, Y: 0, Props: map[string]interface{}{"layer": 0, "prefab": "enemy.yaml"}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy"}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: 0, HoveredIndex: -1, Dragging: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})

	NewEditorEntitySystem().Update(w)
	_, input, _ := rawInputState(w)
	_, pointer, _ := pointerState(w)
	input.LeftDown = true
	pointer.CellX = 5
	pointer.CellY = 1
	NewEditorEntitySystem().Update(w)

	_, undo, _ := undoState(w)
	if len(undo.Snapshots) != 1 {
		t.Fatalf("expected one drag snapshot, got %d", len(undo.Snapshots))
	}
	_, entities, _ := entitiesState(w)
	if entities.Items[0].X != 5*TileSize || entities.Items[0].Y != 1*TileSize {
		t.Fatalf("expected dragged entity at (%d,%d), got (%d,%d)", 5*TileSize, 1*TileSize, entities.Items[0].X, entities.Items[0].Y)
	}
}

func TestEditorEntitySystemDragPreservesPointerOffsetWhileHeld(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{LeftJustPressed: true, LeftDown: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, CellX: 2, CellY: 1, WorldX: 80, WorldY: 48})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", X: TileSize, Y: TileSize, Props: map[string]interface{}{"layer": 0, "prefab": "enemy.yaml"}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Preview: editorio.PrefabPreview{FrameW: 64, FrameH: 32}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: -1, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})

	system := NewEditorEntitySystem()
	system.Update(w)

	_, input, _ := rawInputState(w)
	_, pointer, _ := pointerState(w)
	input.LeftJustPressed = false
	input.LeftDown = true
	pointer.CellX = 3
	pointer.CellY = 1
	pointer.WorldX = 112
	pointer.WorldY = 48
	system.Update(w)

	_, entities, _ := entitiesState(w)
	if entities.Items[0].X != 2*TileSize || entities.Items[0].Y != TileSize {
		t.Fatalf("expected drag to preserve one-cell pointer offset at (%d,%d), got (%d,%d)", 2*TileSize, TileSize, entities.Items[0].X, entities.Items[0].Y)
	}

	pointer.CellX = 4
	pointer.WorldX = 144
	system.Update(w)
	if entities.Items[0].X != 3*TileSize || entities.Items[0].Y != TileSize {
		t.Fatalf("expected held drag to continue moving across frames to (%d,%d), got (%d,%d)", 3*TileSize, TileSize, entities.Items[0].X, entities.Items[0].Y)
	}
}

func TestEditorEntitySystemHiddenLayerEntitiesAreNotHovered(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, WorldX: 16, WorldY: 16})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", X: 0, Y: 0, Props: map[string]interface{}{"layer": 1, "prefab": "enemy.yaml"}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Preview: editorio.PrefabPreview{FrameW: 32, FrameH: 32}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: -1, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})

	visibleLayer := ecs.CreateEntity(w)
	_ = ecs.Add(w, visibleLayer, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Visible", Order: 0, Tiles: make([]int, 400), TilesetUsage: make([]*levels.TileInfo, 400)})
	hiddenLayer := ecs.CreateEntity(w)
	_ = ecs.Add(w, hiddenLayer, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Hidden", Order: 1, Hidden: true, Tiles: make([]int, 400), TilesetUsage: make([]*levels.TileInfo, 400)})

	NewEditorEntitySystem().Update(w)

	_, selection, _ := entitySelectionState(w)
	if selection.HoveredIndex != -1 {
		t.Fatalf("expected hidden-layer entity to be ignored, got hovered index %d", selection.HoveredIndex)
	}
}

func TestEditorEntitySystemIgnoresEntitiesOnOtherVisibleLayers(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{InCanvas: true, HasCell: true, WorldX: 16, WorldY: 16})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", X: 0, Y: 0, Props: map[string]interface{}{"layer": 1, "prefab": "enemy.yaml"}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Preview: editorio.PrefabPreview{FrameW: 32, FrameH: 32}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: -1, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})

	baseLayer := ecs.CreateEntity(w)
	_ = ecs.Add(w, baseLayer, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 1", Order: 0, Tiles: make([]int, 400), TilesetUsage: make([]*levels.TileInfo, 400)})
	upperLayer := ecs.CreateEntity(w)
	_ = ecs.Add(w, upperLayer, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Layer 2", Order: 1, Tiles: make([]int, 400), TilesetUsage: make([]*levels.TileInfo, 400)})

	NewEditorEntitySystem().Update(w)

	_, selection, _ := entitySelectionState(w)
	if selection.HoveredIndex != -1 {
		t.Fatalf("expected off-layer entity to be ignored, got hovered index %d", selection.HoveredIndex)
	}
	if selection.SelectedIndex != -1 {
		t.Fatalf("expected no selected entity, got %d", selection.SelectedIndex)
	}
	_, actions, _ := actionState(w)
	actions.SelectEntity = 0
	NewEditorEntitySystem().Update(w)
	if selection.SelectedIndex != -1 {
		t.Fatalf("expected UI selection on another layer to be ignored, got %d", selection.SelectedIndex)
	}
}

func TestEditorEntitySystemCopyPasteDuplicatesSelectedEntity(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{ID: "enemy_1", Type: "enemy", X: 32, Y: 64, Props: map[string]interface{}{"layer": 0, "prefab": "enemy.yaml", "components": map[string]interface{}{"color": map[string]interface{}{"hex": "#ff0000"}}}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy"}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: 0, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntityClipboardComponent.Kind(), &editorcomponent.EntityClipboardState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1, CopySelectedEntity: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})

	system := NewEditorEntitySystem()
	system.Update(w)
	_, actions, _ := actionState(w)
	actions.PasteCopiedEntity = true
	system.Update(w)

	_, entities, _ := entitiesState(w)
	if len(entities.Items) != 2 {
		t.Fatalf("expected duplicated entity, got %d entities", len(entities.Items))
	}
	duplicate := entities.Items[1]
	if duplicate.Type != "enemy" || duplicate.X != 32 || duplicate.Y != 64 {
		t.Fatalf("expected pasted entity to match source, got %+v", duplicate)
	}
	if duplicate.ID == "enemy_1" {
		t.Fatalf("expected pasted entity id to be uniquified, got %q", duplicate.ID)
	}
	components, ok := duplicate.Props["components"].(map[string]interface{})
	if !ok {
		t.Fatal("expected pasted entity component overrides to be preserved")
	}
	color, ok := components["color"].(map[string]interface{})
	if !ok || color["hex"] != "#ff0000" {
		t.Fatalf("expected pasted overrides to match source, got %+v", components)
	}
	_, selection, _ := entitySelectionState(w)
	if selection.SelectedIndex != 1 {
		t.Fatalf("expected pasted entity to become selected, got %d", selection.SelectedIndex)
	}
}

func TestEditorEntitySystemApplyInspectorDocumentUpdatesEntityAndStatus(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", X: 32, Y: 64, Props: map[string]interface{}{"prefab": "enemy.yaml", "layer": 0}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Components: map[string]any{"transform": map[string]any{"x": 32.0, "y": 64.0}, "color": map[string]any{"hex": "#00ff00"}}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: 0, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1, InspectorDocument: "transform:\n  x: 96\n  y: 160\ncolor:\n  hex: '#ff0000'", ApplyInspectorDocument: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})

	NewEditorEntitySystem().Update(w)

	_, session, _ := sessionState(w)
	if session.Status != "Updated entity component overrides" {
		t.Fatalf("expected success status, got %q", session.Status)
	}
	if !session.Dirty {
		t.Fatal("expected inspector apply to mark the level dirty")
	}
	_, entities, _ := entitiesState(w)
	if entities.Items[0].X != 96 || entities.Items[0].Y != 160 {
		t.Fatalf("expected entity position to update from inspector document, got (%d,%d)", entities.Items[0].X, entities.Items[0].Y)
	}
	components := entityComponentOverrides(entities.Items[0].Props)
	if !inspectorValuesEqual(components, map[string]any{"transform": map[string]any{"x": 96, "y": 160}, "color": map[string]any{"hex": "#ff0000"}}) {
		t.Fatalf("expected overrides to be written, got %+v", components)
	}
	_, actions, _ := actionState(w)
	if actions.ApplyInspectorDocument || actions.InspectorDocument != "" {
		t.Fatalf("expected inspector apply action to be cleared, got %+v", actions)
	}
}

func TestEditorEntitySystemApplyInspectorDocumentReportsParseFailure(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", X: 32, Y: 64, Props: map[string]interface{}{"prefab": "enemy.yaml", "layer": 0, "components": map[string]interface{}{"color": map[string]interface{}{"hex": "#ff0000"}}}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Components: map[string]any{"color": map[string]any{"hex": "#00ff00"}}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: 0, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1, InspectorDocument: "color: [", ApplyInspectorDocument: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 20, Height: 20})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})

	NewEditorEntitySystem().Update(w)

	_, session, _ := sessionState(w)
	if !strings.HasPrefix(session.Status, "Inspector apply failed: ") {
		t.Fatalf("expected parse failure status, got %q", session.Status)
	}
	if session.Dirty {
		t.Fatal("expected failed inspector apply not to mark the level dirty")
	}
	_, entities, _ := entitiesState(w)
	components := entityComponentOverrides(entities.Items[0].Props)
	if !inspectorValuesEqual(components, map[string]any{"color": map[string]any{"hex": "#ff0000"}}) {
		t.Fatalf("expected failed apply to preserve entity overrides, got %+v", components)
	}
	_, actions, _ := actionState(w)
	if actions.ApplyInspectorDocument || actions.InspectorDocument != "" {
		t.Fatalf("expected inspector apply action to be cleared after failure, got %+v", actions)
	}
}
