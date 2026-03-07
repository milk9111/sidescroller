package editorsystem

import (
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestEditorLayerSystemMoveRemapsEntityLayers(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 4, Height: 4})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", Props: map[string]interface{}{"layer": 0}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, MoveLayerDelta: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})

	for index := 0; index < 3; index++ {
		entity := ecs.CreateEntity(w)
		_ = ecs.Add(w, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: string(rune('A' + index)), Order: index, Tiles: make([]int, 16), TilesetUsage: make([]*levels.TileInfo, 16)})
	}

	NewEditorLayerSystem().Update(w)

	_, session, _ := sessionState(w)
	if session.CurrentLayer != 1 {
		t.Fatalf("expected current layer 1, got %d", session.CurrentLayer)
	}
	_, entities, _ := entitiesState(w)
	if got, ok := entityLayerIndex(entities.Items[0].Props); !ok || got != 1 {
		t.Fatalf("expected entity layer to remap to 1, got %v (ok=%t)", got, ok)
	}
	layers := layerEntities(w)
	first, _ := ecs.Get(w, layers[0], editorcomponent.LayerDataComponent.Kind())
	second, _ := ecs.Get(w, layers[1], editorcomponent.LayerDataComponent.Kind())
	if first.Name != "B" || second.Name != "A" {
		t.Fatalf("expected layers to swap order, got %s then %s", first.Name, second.Name)
	}
}

func TestEditorLayerSystemTogglesVisibilityWithoutDirtyingLevel(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 4, Height: 4})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, ToggleLayerVisibility: true})

	layerEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "A", Order: 0, Tiles: make([]int, 16), TilesetUsage: make([]*levels.TileInfo, 16)})

	NewEditorLayerSystem().Update(w)

	_, session, _ := sessionState(w)
	layer, _ := ecs.Get(w, layerEntity, editorcomponent.LayerDataComponent.Kind())
	if !layer.Hidden {
		t.Fatalf("expected layer to be hidden")
	}
	if session.Dirty {
		t.Fatalf("expected cosmetic visibility toggle to leave session clean")
	}
	if session.Status != "Layer hidden" {
		t.Fatalf("expected hidden status, got %q", session.Status)
	}
	_, actions, _ := actionState(w)
	if actions.ToggleLayerVisibility {
		t.Fatalf("expected toggle action to be cleared")
	}
	if layerVisible(layer) {
		t.Fatalf("expected helper visibility to report false")
	}
}

func TestEditorLayerSystemDeletesSelectedLayerAndRemapsEntities(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 4, Height: 4})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{
		{Type: "enemy", Props: map[string]interface{}{"layer": 0}},
		{Type: "enemy", Props: map[string]interface{}{"layer": 1}},
		{Type: "enemy", Props: map[string]interface{}{"layer": 2}},
	}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: 1, HoveredIndex: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, DeleteCurrentLayer: true})
	_ = ecs.Add(w, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})
	_ = ecs.Add(w, sessionEntity, editorcomponent.OverviewStateComponent.Kind(), &editorcomponent.OverviewState{})
	_ = ecs.Add(w, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{
		DirtyCells:  map[int]map[int]struct{}{1: {3: {}}, 2: {4: {}}},
		FullRebuild: map[int]bool{1: true, 2: true},
	})

	for index := 0; index < 3; index++ {
		entity := ecs.CreateEntity(w)
		_ = ecs.Add(w, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: string(rune('A' + index)), Order: index, Tiles: make([]int, 16), TilesetUsage: make([]*levels.TileInfo, 16)})
	}

	NewEditorLayerSystem().Update(w)

	_, session, _ := sessionState(w)
	if session.CurrentLayer != 1 {
		t.Fatalf("expected current layer to stay on remapped index 1, got %d", session.CurrentLayer)
	}
	if session.Status != "Deleted layer" {
		t.Fatalf("expected deleted status, got %q", session.Status)
	}
	if !session.Dirty {
		t.Fatalf("expected layer delete to dirty session")
	}
	layers := layerEntities(w)
	if len(layers) != 2 {
		t.Fatalf("expected 2 layers remaining, got %d", len(layers))
	}
	first, _ := ecs.Get(w, layers[0], editorcomponent.LayerDataComponent.Kind())
	second, _ := ecs.Get(w, layers[1], editorcomponent.LayerDataComponent.Kind())
	if first.Name != "A" || second.Name != "C" {
		t.Fatalf("expected layers A and C to remain, got %s and %s", first.Name, second.Name)
	}
	_, entities, _ := entitiesState(w)
	if len(entities.Items) != 2 {
		t.Fatalf("expected entities on deleted layer to be removed, got %d entities", len(entities.Items))
	}
	if got, ok := entityLayerIndex(entities.Items[0].Props); !ok || got != 0 {
		t.Fatalf("expected first entity to remain on layer 0, got %v (ok=%t)", got, ok)
	}
	if got, ok := entityLayerIndex(entities.Items[1].Props); !ok || got != 1 {
		t.Fatalf("expected upper entity to shift to layer 1, got %v (ok=%t)", got, ok)
	}
	_, selection, _ := entitySelectionState(w)
	if selection.SelectedIndex != -1 || selection.HoveredIndex != -1 {
		t.Fatalf("expected entity selection to clear after layer delete")
	}
	_, autotile, _ := autotileState(w)
	if _, exists := autotile.DirtyCells[1]; !exists {
		t.Fatalf("expected higher dirty autotile cells to shift down")
	}
	if _, exists := autotile.DirtyCells[2]; exists {
		t.Fatalf("expected deleted autotile layer indices to be removed")
	}
	if !autotile.FullRebuild[1] {
		t.Fatalf("expected higher full rebuild flag to shift down")
	}
	_, actions, _ := actionState(w)
	if actions.DeleteCurrentLayer {
		t.Fatalf("expected delete layer action to be cleared")
	}
}

func TestEditorLayerSystemDoesNotDeleteLastLayer(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 4, Height: 4})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, DeleteCurrentLayer: true})

	layerEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, layerEntity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Only", Order: 0, Tiles: make([]int, 16), TilesetUsage: make([]*levels.TileInfo, 16)})

	NewEditorLayerSystem().Update(w)

	layers := layerEntities(w)
	if len(layers) != 1 {
		t.Fatalf("expected last layer to remain, got %d layers", len(layers))
	}
	_, session, _ := sessionState(w)
	if session.Status != "Cannot delete last layer" {
		t.Fatalf("expected last-layer guard status, got %q", session.Status)
	}
	if session.Dirty {
		t.Fatalf("expected rejected delete to leave session clean")
	}
}
