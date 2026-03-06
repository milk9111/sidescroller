package editorsystem

import (
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
