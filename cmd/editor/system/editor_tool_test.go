package editorsystem

import (
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestApplySpikeAtCreatesAndReusesSpikeEntity(t *testing.T) {
	w := ecs.NewWorld()
	entity := ecs.CreateEntity(w)
	_ = ecs.Add(w, entity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 1})
	_ = ecs.Add(w, entity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{Width: 4, Height: 4})
	_ = ecs.Add(w, entity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{})
	physicsLayer := ecs.CreateEntity(w)
	tiles := make([]int, 16)
	tiles[2*4+1] = 1
	_ = ecs.Add(w, physicsLayer, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Physics", Order: 0, Physics: true, Tiles: tiles, TilesetUsage: make([]*levels.TileInfo, 16)})

	tool := NewEditorToolSystem()
	_, session, _ := sessionState(w)
	_, meta, _ := levelMetaState(w)
	if !tool.applySpikeAt(w, session, meta, 1, 1) {
		t.Fatal("expected first spike placement to create entity")
	}
	_, entities, _ := entitiesState(w)
	if len(entities.Items) != 1 {
		t.Fatalf("expected one spike entity, got %d", len(entities.Items))
	}
	placed := entities.Items[0]
	if !isSpikeEntity(placed) {
		t.Fatalf("expected spike type, got %q", placed.Type)
	}
	if got := toFloat(placed.Props["rotation"]); got != 0 {
		t.Fatalf("expected upward spike rotation 0, got %v", got)
	}
	if got, ok := entityLayerIndex(placed.Props); !ok || got != 1 {
		t.Fatalf("expected layer 1, got %d (ok=%t)", got, ok)
	}

	session.CurrentLayer = 2
	if tool.applySpikeAt(w, session, meta, 1, 1) {
		if len(entities.Items) != 1 {
			t.Fatalf("expected existing spike to be reused, got %d entities", len(entities.Items))
		}
	}
	if got, ok := entityLayerIndex(entities.Items[0].Props); !ok || got != 2 {
		t.Fatalf("expected reused spike layer 2, got %d (ok=%t)", got, ok)
	}
}
