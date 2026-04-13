package savegame

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveRootDirForWindowsAndLinux(t *testing.T) {
	linuxDir, err := saveRootDirFor("linux", "/home/connor")
	if err != nil {
		t.Fatalf("linux save root: %v", err)
	}
	if linuxDir != filepath.Join("/home/connor", ".local", "share", "milk9111", "Defective") {
		t.Fatalf("unexpected linux save root %q", linuxDir)
	}

	windowsDir, err := saveRootDirFor("windows", `C:\Users\Connor`)
	if err != nil {
		t.Fatalf("windows save root: %v", err)
	}
	if windowsDir != filepath.Join(`C:\Users\Connor`, "AppData", "LocalLow", "milk9111", "Defective") {
		t.Fatalf("unexpected windows save root %q", windowsDir)
	}
}

func TestNormalizeFileName(t *testing.T) {
	name, err := normalizeFileName("slot_one")
	if err != nil {
		t.Fatalf("normalize file name: %v", err)
	}
	if name != "slot_one.json" {
		t.Fatalf("expected .json extension, got %q", name)
	}

	if _, err := normalizeFileName("profiles/slot_one.json"); err == nil {
		t.Fatal("expected path input to be rejected")
	}
}

func TestIsWebTarget(t *testing.T) {
	if !isWebTarget("js", "wasm") {
		t.Fatal("expected js/wasm to be treated as web target")
	}
	if isWebTarget("linux", "amd64") {
		t.Fatal("did not expect linux/amd64 to be treated as web target")
	}
}

func TestStoreSaveAndLoadRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	store := &Store{path: filepath.Join(tmpDir, "slot.json")}
	snapshot := &File{
		Version: 1,
		Level:   "long_fall.json",
		Player: PlayerState{
			Health:             HealthState{Initial: 3, Current: 2},
			Inventory:          []InventoryItem{{Prefab: "item_wrench.yaml", Count: 1}},
			TransitionCooldown: &TransitionCooldownState{Active: true, TransitionID: "right", TransitionIDs: []string{"right"}},
			TransitionPop:      &TransitionPopState{VX: 2, VY: -4, FacingLeft: true, WallJumpDur: 4, WallJumpX: -10},
		},
		LevelLayerStates:  map[string]bool{"long_fall.json#secret_layer": false},
		LevelEntityStates: map[string]string{"long_fall.json#trigger_1": "used"},
	}

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if loaded.Level != snapshot.Level {
		t.Fatalf("expected level %q, got %q", snapshot.Level, loaded.Level)
	}
	if loaded.Player.Health.Current != 2 {
		t.Fatalf("expected current health 2, got %d", loaded.Player.Health.Current)
	}
	if len(loaded.Player.Inventory) != 1 || loaded.Player.Inventory[0].Prefab != "item_wrench.yaml" {
		t.Fatalf("unexpected inventory %+v", loaded.Player.Inventory)
	}
	if loaded.LevelEntityStates["long_fall.json#trigger_1"] != "used" {
		t.Fatalf("unexpected level state map %+v", loaded.LevelEntityStates)
	}
	if loaded.Player.TransitionCooldown == nil || !loaded.Player.TransitionCooldown.Active || loaded.Player.TransitionCooldown.TransitionID != "right" {
		t.Fatalf("unexpected transition cooldown %+v", loaded.Player.TransitionCooldown)
	}
	if loaded.Player.TransitionPop == nil || loaded.Player.TransitionPop.VY != -4 || !loaded.Player.TransitionPop.FacingLeft {
		t.Fatalf("unexpected transition pop %+v", loaded.Player.TransitionPop)
	}
	if active, ok := loaded.LevelLayerStates["long_fall.json#secret_layer"]; !ok || active {
		t.Fatalf("unexpected level layer state map %+v", loaded.LevelLayerStates)
	}

	data, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatal("expected trailing newline in save file")
	}
}

func TestDisabledStoreSkipsLoadAndSave(t *testing.T) {
	store := &Store{disabled: true, path: filepath.Join(t.TempDir(), "slot.json")}
	snapshot := &File{Level: "long_fall.json"}

	if err := store.Save(snapshot); err != nil {
		t.Fatalf("save should be a no-op for disabled store: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load should be a no-op for disabled store: %v", err)
	}
	if loaded != nil {
		t.Fatalf("expected nil loaded save for disabled store, got %+v", loaded)
	}

	if _, err := os.Stat(store.path); !os.IsNotExist(err) {
		t.Fatalf("expected disabled store to avoid creating a save file, got err=%v", err)
	}
}
