package editorsystem

import (
	"os"
	"path/filepath"
	"testing"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
	"gopkg.in/yaml.v3"
)

func TestEditorPrefabSystemCreatesPrefabAndRebindsEntity(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "prefabs"), 0o755); err != nil {
		t.Fatalf("mkdir prefabs: %v", err)
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	w := ecs.NewWorld()
	sessionEntity := ecs.CreateEntity(w)
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{CurrentLayer: 0})
	_ = ecs.Add(w, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: []levels.Entity{{Type: "enemy", X: 64, Y: 96, Props: map[string]interface{}{"layer": 0, "prefab": "enemy.yaml", "components": map[string]interface{}{"color": map[string]interface{}{"hex": "#00ff00"}}}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: 0, HoveredIndex: -1})
	_ = ecs.Add(w, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: []editorio.PrefabInfo{{Name: "enemy", Path: "enemy.yaml", EntityType: "enemy", Components: map[string]any{"sprite": map[string]any{"image": "enemy.png"}, "transform": map[string]any{"scale_x": 1.0, "scale_y": 1.0}}}}})
	_ = ecs.Add(w, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1, SelectEntity: -1, ConvertSelectedEntityToPrefabName: "enemy_variant", ApplyConvertSelectedEntityToPrefab: true})

	NewEditorPrefabSystem(root, "prefabs").Update(w)

	data, err := os.ReadFile(filepath.Join(root, "prefabs", "enemy_variant.yaml"))
	if err != nil {
		t.Fatalf("read prefab: %v", err)
	}
	var saved struct {
		Name       string         `yaml:"name"`
		Components map[string]any `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("unmarshal prefab: %v", err)
	}
	if saved.Name != "enemy" {
		t.Fatalf("expected prefab entity type name enemy, got %q", saved.Name)
	}
	color, ok := saved.Components["color"].(map[string]interface{})
	if !ok || color["hex"] != "#00ff00" {
		t.Fatalf("expected merged color override in prefab, got %+v", saved.Components)
	}
	_, entities, _ := entitiesState(w)
	if prefabPath, _ := entities.Items[0].Props["prefab"].(string); prefabPath != "enemy_variant.yaml" {
		t.Fatalf("expected entity prefab to be rebound, got %q", prefabPath)
	}
	if _, ok := entities.Items[0].Props["components"]; ok {
		t.Fatal("expected instance component overrides to be cleared after conversion")
	}
	_, catalog, _ := prefabCatalogState(w)
	found := false
	for _, item := range catalog.Items {
		if item.Path == "enemy_variant.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected prefab catalog to include the new prefab")
	}
}
