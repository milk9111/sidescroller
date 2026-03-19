package editorio

import (
	"testing"

	"github.com/milk9111/sidescroller/prefabs"
)

func TestPreviewTintFromColorHexAndOverrides(t *testing.T) {
	g := 0.5
	spec := prefabs.ColorComponentSpec{
		Hex: "#804020C0",
		G:   &g,
	}

	tint, ok := previewTintFromColor(spec)
	if !ok {
		t.Fatal("expected tint to be detected")
	}
	if tint.r != 128.0/255.0 {
		t.Fatalf("expected red from hex, got %f", tint.r)
	}
	if tint.g != 0.5 {
		t.Fatalf("expected green override 0.5, got %f", tint.g)
	}
	if tint.b != 32.0/255.0 {
		t.Fatalf("expected blue from hex, got %f", tint.b)
	}
	if tint.a != 192.0/255.0 {
		t.Fatalf("expected alpha from hex, got %f", tint.a)
	}
}

func TestPreviewTintFromColorEmptySpec(t *testing.T) {
	tint, ok := previewTintFromColor(prefabs.ColorComponentSpec{})
	if ok {
		t.Fatal("expected empty color spec to report no tint")
	}
	if tint.r != 1 || tint.g != 1 || tint.b != 1 || tint.a != 1 {
		t.Fatalf("expected default identity tint, got %+v", tint)
	}
}

func TestLoadPrefabInfoPreservesTintWhenPreviewComesFromBuildSpec(t *testing.T) {
	info, err := loadPrefabInfo("garbage_heap_big_green.yaml")
	if err != nil {
		t.Fatalf("load prefab info: %v", err)
	}
	if !info.Preview.HasTint {
		t.Fatal("expected prefab tint to be preserved")
	}
	if info.Preview.TintR == 1 && info.Preview.TintG == 1 && info.Preview.TintB == 1 {
		t.Fatalf("expected non-identity tint, got %+v", info.Preview)
	}
	if info.Preview.ImagePath == "" {
		t.Fatal("expected preview image from inherited build spec")
	}
}

func TestLoadPrefabInfoCarriesTransformScaleIntoPreview(t *testing.T) {
	info, err := loadPrefabInfo("claw_pickup.yaml")
	if err != nil {
		t.Fatalf("load prefab info: %v", err)
	}
	if info.Preview.ScaleX != 0.25 || info.Preview.ScaleY != 0.25 {
		t.Fatalf("expected preview scale 0.25/0.25, got %f/%f", info.Preview.ScaleX, info.Preview.ScaleY)
	}
}

func TestLoadPrefabInfoUsesExplicitEntityTypeWhenProvided(t *testing.T) {
	info, err := loadPrefabInfo("breakable_cracks.yaml")
	if err != nil {
		t.Fatalf("load prefab info: %v", err)
	}
	if info.Name != "breakable_cracks" {
		t.Fatalf("expected prefab display name breakable_cracks, got %q", info.Name)
	}
	if info.EntityType != "breakable_wall" {
		t.Fatalf("expected explicit entity type breakable_wall, got %q", info.EntityType)
	}
}

func TestResolvePrefabPreviewIgnoresIrrelevantOverridesWithoutHeavyAllocations(t *testing.T) {
	info := PrefabInfo{
		Preview: PrefabPreview{
			ImagePath:    "player_v3-Sheet.png",
			FrameW:       64,
			FrameH:       64,
			FallbackSize: 64,
			ScaleX:       1,
			ScaleY:       1,
			RenderLayer:  100,
		},
		Components: map[string]any{
			"transform": map[string]any{"scale_x": 1.0, "scale_y": 1.0},
			"sprite":    map[string]any{"image": "player_v3-Sheet.png"},
			"animation": map[string]any{
				"sheet":   "player_v3-Sheet.png",
				"current": "idle",
				"defs": map[string]any{
					"idle": map[string]any{"row": 0, "col_start": 0, "frame_w": 64, "frame_h": 64, "frame_count": 5},
					"run":  map[string]any{"row": 1, "col_start": 0, "frame_w": 64, "frame_h": 64, "frame_count": 8},
				},
			},
			"audio": map[string]any{
				"clips": []interface{}{
					map[string]any{"name": "run", "file": "player_rolling.wav"},
					map[string]any{"name": "jump", "file": "player_jump.wav"},
				},
			},
			"hitboxes": []interface{}{
				map[string]any{"width": 60, "height": 24, "frames": []interface{}{4}},
				map[string]any{"width": 24, "height": 60, "frames": []interface{}{5}},
			},
		},
	}
	overrides := map[string]any{
		"persistent": map[string]any{"id": "player"},
		"player":     map[string]any{"move_speed": 4.0},
	}

	allocs := testing.AllocsPerRun(100, func() {
		preview := ResolvePrefabPreview(info, overrides)
		if preview.ImagePath != info.Preview.ImagePath {
			t.Fatalf("expected cached preview image %q, got %q", info.Preview.ImagePath, preview.ImagePath)
		}
	})
	if allocs > 1 {
		t.Fatalf("expected preview resolution with irrelevant overrides to avoid heap churn, got %.2f allocs/run", allocs)
	}
	preview := ResolvePrefabPreview(info, overrides)
	if preview.FrameW != info.Preview.FrameW || preview.FrameH != info.Preview.FrameH {
		t.Fatalf("expected cached preview frame %dx%d, got %dx%d", info.Preview.FrameW, info.Preview.FrameH, preview.FrameW, preview.FrameH)
	}
}

func TestResolvePrefabPreviewAppliesRelevantTransformOverride(t *testing.T) {
	info := PrefabInfo{
		Preview: PrefabPreview{ScaleX: 1, ScaleY: 1, FallbackSize: 32},
		Components: map[string]any{
			"transform": map[string]any{"scale_x": 1.0, "scale_y": 1.0},
		},
	}

	preview := ResolvePrefabPreview(info, map[string]any{
		"transform": map[string]any{"scale_x": 2.0, "scale_y": 0.5},
	})
	if preview.ScaleX != 2 || preview.ScaleY != 0.5 {
		t.Fatalf("expected transform override scale 2/0.5, got %f/%f", preview.ScaleX, preview.ScaleY)
	}
}
