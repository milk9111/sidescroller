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
