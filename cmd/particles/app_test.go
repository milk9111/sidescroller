package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milk9111/sidescroller/prefabs"
	"gopkg.in/yaml.v3"
)

func TestBuildEmitterPrefabIncludesTransformAndEmitter(t *testing.T) {
	emitter := prefabs.ParticleEmitterComponentSpec{
		TotalParticles: 24,
		Lifetime:       18,
		Burst:          true,
		Continuous:     false,
		HasGravity:     true,
		Image:          "basic_particle.png",
		Color:          "#AABBCC",
	}

	spec := buildEmitterPrefab("sparks", emitter)
	if spec.Name != "sparks" {
		t.Fatalf("expected prefab name to match, got %q", spec.Name)
	}
	if _, ok := spec.Components["transform"]; !ok {
		t.Fatal("expected transform component to be present")
	}
	rawEmitter, ok := spec.Components["particle_emitter"]
	if !ok {
		t.Fatal("expected particle_emitter component to be present")
	}
	decoded, err := prefabs.DecodeComponentSpec[prefabs.ParticleEmitterComponentSpec](rawEmitter)
	if err != nil {
		t.Fatalf("decode particle emitter: %v", err)
	}
	if decoded.TotalParticles != emitter.TotalParticles || decoded.Image != emitter.Image || decoded.Color != emitter.Color {
		t.Fatalf("decoded particle emitter mismatch: %#v", decoded)
	}
	transform, err := prefabs.DecodeComponentSpec[prefabs.TransformComponentSpec](spec.Components["transform"])
	if err != nil {
		t.Fatalf("decode transform: %v", err)
	}
	if transform.ScaleX != 1 || transform.ScaleY != 1 {
		t.Fatalf("expected default scale 1, got %#v", transform)
	}
}

func TestSaveEmitterPrefabWritesYamlToPrefabDir(t *testing.T) {
	workspace := t.TempDir()
	spec := buildEmitterPrefab("embers", prefabs.ParticleEmitterComponentSpec{
		TotalParticles: 12,
		Lifetime:       22,
		Burst:          false,
		Continuous:     true,
		HasGravity:     true,
		Image:          "basic_particle.png",
		Color:          "#FFFFFF",
	})

	path, err := saveEmitterPrefab(workspace, "prefabs", "embers", spec)
	if err != nil {
		t.Fatalf("save emitter prefab: %v", err)
	}
	if filepath.ToSlash(path) != filepath.ToSlash(filepath.Join(workspace, "prefabs", "embers.yaml")) {
		t.Fatalf("unexpected save path %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved prefab: %v", err)
	}
	text := string(data)
	for _, want := range []string{"name: embers", "transform:", "particle_emitter:", "total_particles: 12", "image: basic_particle.png"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected saved YAML to contain %q, got:\n%s", want, text)
		}
	}
}

func TestParsePositiveIntRejectsNonPositiveValues(t *testing.T) {
	if _, err := parsePositiveInt("0", "total_particles"); err == nil {
		t.Fatal("expected zero to be rejected")
	}
	if _, err := parsePositiveInt("abc", "total_particles"); err == nil {
		t.Fatal("expected non-numeric input to be rejected")
	}
}

func TestParseNRGBAAcceptsEightDigitHex(t *testing.T) {
	parsed, err := parseNRGBA("#11223344")
	if err != nil {
		t.Fatalf("parse color: %v", err)
	}
	if parsed.R != 0x11 || parsed.G != 0x22 || parsed.B != 0x33 || parsed.A != 0x44 {
		t.Fatalf("unexpected parsed color: %#v", parsed)
	}
}

func TestLoadEmitterPrefabPopulatesEmitterForm(t *testing.T) {
	workspace := t.TempDir()
	prefabDir := filepath.Join(workspace, "prefabs")
	if err := os.MkdirAll(prefabDir, 0o755); err != nil {
		t.Fatalf("create prefab dir: %v", err)
	}
	spec := buildEmitterPrefab("embers", prefabs.ParticleEmitterComponentSpec{
		TotalParticles: 48,
		Lifetime:       12,
		Burst:          false,
		Continuous:     true,
		HasGravity:     true,
		Image:          "basic_particle.png",
		Color:          "#ABCDEF",
	})
	data, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal prefab: %v", err)
	}
	if err := os.WriteFile(filepath.Join(prefabDir, "embers.yaml"), data, 0o644); err != nil {
		t.Fatalf("write prefab: %v", err)
	}

	loaded, err := loadEmitterPrefab(workspace, "prefabs", "embers")
	if err != nil {
		t.Fatalf("load emitter prefab: %v", err)
	}
	if loaded.normalizedPath != "embers.yaml" {
		t.Fatalf("expected normalized path embers.yaml, got %q", loaded.normalizedPath)
	}
	if loaded.form.Name != "embers" {
		t.Fatalf("expected form name embers, got %q", loaded.form.Name)
	}
	if loaded.form.TotalParticles != "48" || loaded.form.Lifetime != "12" {
		t.Fatalf("unexpected numeric fields: %#v", loaded.form)
	}
	if !loaded.form.Continuous || !loaded.form.HasGravity || loaded.form.Burst {
		t.Fatalf("unexpected boolean fields: %#v", loaded.form)
	}
	if loaded.form.Image != "basic_particle.png" || loaded.form.Color != "#ABCDEF" {
		t.Fatalf("unexpected asset fields: %#v", loaded.form)
	}
}

func TestLoadEmitterPrefabRejectsMissingParticleEmitter(t *testing.T) {
	workspace := t.TempDir()
	prefabDir := filepath.Join(workspace, "prefabs")
	if err := os.MkdirAll(prefabDir, 0o755); err != nil {
		t.Fatalf("create prefab dir: %v", err)
	}
	spec := prefabs.EntityBuildSpec{
		Name: "not_an_emitter",
		Components: map[string]any{
			"transform": prefabs.TransformComponentSpec{ScaleX: 1, ScaleY: 1},
		},
	}
	data, err := yaml.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal prefab: %v", err)
	}
	if err := os.WriteFile(filepath.Join(prefabDir, "not_an_emitter.yaml"), data, 0o644); err != nil {
		t.Fatalf("write prefab: %v", err)
	}

	if _, err := loadEmitterPrefab(workspace, "prefabs", "not_an_emitter"); err == nil {
		t.Fatal("expected missing particle_emitter component to fail")
	}
}
