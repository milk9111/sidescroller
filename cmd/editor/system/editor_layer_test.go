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
