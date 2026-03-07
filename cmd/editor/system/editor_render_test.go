package editorsystem

import (
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

func TestSelectedPrefabPreviewUsesCurrentPlacementState(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 2})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{HasCell: true, CellX: 3, CellY: 4})
	selected := editorio.PrefabInfo{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Preview: editorio.PrefabPreview{RenderLayer: 12}}
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{selected}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{SelectedPath: "enemy.yaml", SelectedType: "enemy"})

	item, prefab, ok := selectedPrefabPreview(w)
	if !ok {
		t.Fatal("expected selected prefab preview to be available")
	}
	if item.Type != "enemy" {
		t.Fatalf("expected preview type enemy, got %q", item.Type)
	}
	if item.X != 3*TileSize || item.Y != 4*TileSize {
		t.Fatalf("expected preview position (%d,%d), got (%d,%d)", 3*TileSize, 4*TileSize, item.X, item.Y)
	}
	if layer, ok := entityLayerIndex(item.Props); !ok || layer != 2 {
		t.Fatalf("expected preview layer 2, got %v (ok=%t)", layer, ok)
	}
	if prefab == nil || prefab.Path != "enemy.yaml" {
		t.Fatalf("expected prefab enemy.yaml, got %+v", prefab)
	}
}

func TestSelectedPrefabPreviewRespectsCurrentLayer(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{HasCell: true, CellX: 1, CellY: 2})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "Enemy", Path: "enemy.yaml", EntityType: "enemy", Preview: editorio.PrefabPreview{RenderLayer: 3}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{SelectedPath: "enemy.yaml", SelectedType: "enemy"})

	item, _, ok := selectedPrefabPreview(w)
	if !ok {
		t.Fatal("expected selected prefab preview to be available")
	}
	if got := normalizedEntityLayerIndex(item); got != 1 {
		t.Fatalf("expected preview to use current layer 1, got %d", got)
	}
}

func TestCurrentLayerOutlineIndicesOnlyIncludeActiveVisibleLayer(t *testing.T) {
	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 1})

	layer0 := ecs.CreateEntity(w)
	_ = ecs.Add(w, layer0, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Base", Order: 0, Tiles: make([]int, 1), TilesetUsage: make([]*levels.TileInfo, 1)})
	layer1 := ecs.CreateEntity(w)
	_ = ecs.Add(w, layer1, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Active", Order: 1, Tiles: make([]int, 1), TilesetUsage: make([]*levels.TileInfo, 1)})
	layer2 := ecs.CreateEntity(w)
	_ = ecs.Add(w, layer2, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{Name: "Hidden", Order: 2, Hidden: true, Tiles: make([]int, 1), TilesetUsage: make([]*levels.TileInfo, 1)})

	items := []levels.Entity{
		{Type: "enemy", Props: map[string]interface{}{"layer": 0}},
		{Type: "enemy", Props: map[string]interface{}{"layer": 1}},
		{Type: "transition", Props: map[string]interface{}{"layer": 1, "w": float64(TileSize), "h": float64(TileSize)}},
		{Type: "enemy", Props: map[string]interface{}{"layer": 2}},
	}

	_, session, _ := sessionState(w)
	indices := currentLayerOutlineIndices(w, session, items)
	if len(indices) != 2 {
		t.Fatalf("expected 2 outlined entities on active visible layer, got %d", len(indices))
	}
	if indices[0] != 1 || indices[1] != 2 {
		t.Fatalf("expected outlined indices [1 2], got %v", indices)
	}
}
