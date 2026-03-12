package editorsystem

import (
	"reflect"
	"testing"

	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/levels"
)

func TestBuildInspectorDocumentReturnsEmptyForNoComponents(t *testing.T) {
	document, err := buildInspectorDocument(nil)
	if err != nil {
		t.Fatalf("buildInspectorDocument() error = %v", err)
	}
	if document != "" {
		t.Fatalf("expected empty document, got %q", document)
	}
}

func TestApplyInspectorDocumentEditWritesOnlyPrefabOverrides(t *testing.T) {
	prefab := &editorio.PrefabInfo{
		Path: "enemy.yaml",
		Components: map[string]any{
			"transform": map[string]any{"x": 1.0, "y": 2.0, "scale_x": 1.0, "scale_y": 1.0},
			"color":     map[string]any{"hex": "#00ff00"},
			"sprite":    map[string]any{"image": "enemy.png"},
		},
	}
	item := &levels.Entity{
		Type: "enemy",
		X:    32,
		Y:    64,
		Props: map[string]interface{}{
			"prefab": "enemy.yaml",
		},
	}
	document := "transform:\n  x: 96\n  y: 160\n  scale_x: 1\n  scale_y: 1\ncolor:\n  hex: '#ff0000'\nsprite:\n  image: enemy.png"

	changed, err := applyInspectorDocumentEdit(item, prefab, document)
	if err != nil {
		t.Fatalf("applyInspectorDocumentEdit() error = %v", err)
	}
	if !changed {
		t.Fatal("expected prefab-backed inspector edit to change the entity")
	}
	if item.X != 96 || item.Y != 160 {
		t.Fatalf("expected entity position to sync from transform, got (%d,%d)", item.X, item.Y)
	}
	overrides := entityComponentOverrides(item.Props)
	if len(overrides) != 2 {
		t.Fatalf("expected only changed component overrides, got %+v", overrides)
	}
	transform, ok := overrides["transform"].(map[string]any)
	if !ok {
		t.Fatalf("expected transform override map, got %#v", overrides["transform"])
	}
	if !inspectorValuesEqual(transform, map[string]any{"x": 96, "y": 160}) {
		t.Fatalf("expected only changed transform fields in overrides, got %+v", transform)
	}
	color, ok := overrides["color"].(map[string]any)
	if !ok || color["hex"] != "#ff0000" {
		t.Fatalf("expected color override to be preserved, got %+v", overrides["color"])
	}
	if _, ok := overrides["sprite"]; ok {
		t.Fatalf("expected unchanged prefab component to stay out of overrides, got %+v", overrides)
	}
	if !reflect.DeepEqual(prefab.Components["color"], map[string]any{"hex": "#00ff00"}) {
		t.Fatalf("expected prefab component data to remain unchanged, got %+v", prefab.Components)
	}
	if got := item.Props["scale_x"]; got != 1.0 {
		t.Fatalf("expected root scale_x to sync from effective transform, got %#v", got)
	}
	if got := item.Props["scale_y"]; got != 1.0 {
		t.Fatalf("expected root scale_y to sync from effective transform, got %#v", got)
	}
}

func TestApplyInspectorDocumentEditRemovesOverridesWhenMatchingPrefab(t *testing.T) {
	prefab := &editorio.PrefabInfo{
		Path: "enemy.yaml",
		Components: map[string]any{
			"transform": map[string]any{"x": 32.0, "y": 64.0, "rotation": 0.0},
			"color":     map[string]any{"hex": "#00ff00"},
		},
	}
	item := &levels.Entity{
		Type: "enemy",
		X:    32,
		Y:    64,
		Props: map[string]interface{}{
			"prefab":     "enemy.yaml",
			"rotation":   45.0,
			"components": map[string]interface{}{"color": map[string]interface{}{"hex": "#ff0000"}, "transform": map[string]interface{}{"rotation": 45.0}},
		},
	}
	document := "transform:\n  x: 32\n  y: 64\n  rotation: 0\ncolor:\n  hex: '#00ff00'"

	changed, err := applyInspectorDocumentEdit(item, prefab, document)
	if err != nil {
		t.Fatalf("applyInspectorDocumentEdit() error = %v", err)
	}
	if !changed {
		t.Fatal("expected matching prefab values to clear stale overrides")
	}
	if overrides := entityComponentOverrides(item.Props); overrides != nil {
		t.Fatalf("expected overrides to be removed when document matches prefab, got %+v", overrides)
	}
	if got := item.Props["rotation"]; got != 0.0 {
		t.Fatalf("expected root rotation prop to normalize to the prefab default, got %+v", item.Props)
	}
	if !inspectorValuesEqual(item.Props["transform"], map[string]any{"x": 32.0, "y": 64.0, "rotation": 0.0}) {
		t.Fatalf("expected legacy transform props to mirror the effective transform, got %+v", item.Props["transform"])
	}
}

func TestApplyInspectorDocumentEditWithoutPrefabStoresWholeDocument(t *testing.T) {
	item := &levels.Entity{Type: "orphan", Props: map[string]interface{}{}}
	document := "script:\n  name: custom\nhitboxes:\n  - x: 1\n    y: 2"

	changed, err := applyInspectorDocumentEdit(item, nil, document)
	if err != nil {
		t.Fatalf("applyInspectorDocumentEdit() error = %v", err)
	}
	if !changed {
		t.Fatal("expected entity-local inspector document to be stored")
	}
	overrides := entityComponentOverrides(item.Props)
	if !inspectorValuesEqual(overrides, map[string]any{
		"script":   map[string]any{"name": "custom"},
		"hitboxes": []any{map[string]any{"x": 1, "y": 2}},
	}) {
		t.Fatalf("expected full document to be stored for non-prefab entity, got %+v", overrides)
	}
}

func TestApplyInspectorDocumentEditRejectsInvalidYAML(t *testing.T) {
	original := levels.Entity{
		Type: "enemy",
		Props: map[string]interface{}{
			"prefab": "enemy.yaml",
			"components": map[string]interface{}{
				"color": map[string]interface{}{"hex": "#ff0000"},
			},
		},
	}
	item := cloneEditorEntity(original)

	changed, err := applyInspectorDocumentEdit(&item, &editorio.PrefabInfo{Path: "enemy.yaml", Components: map[string]any{"color": map[string]any{"hex": "#00ff00"}}}, "color: [")
	if err == nil {
		t.Fatal("expected invalid YAML to fail")
	}
	if changed {
		t.Fatal("expected invalid YAML to leave entity unchanged")
	}
	if !reflect.DeepEqual(item, original) {
		t.Fatalf("expected invalid YAML to avoid mutating the entity, got %+v", item)
	}
}
