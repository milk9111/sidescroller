package savegame

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListSlotsInDirReturnsSortedValidSaves(t *testing.T) {
	root := t.TempDir()
	writeSaveFixture(t, root, "slot_c.json", "boss_room.json", 3)
	writeSaveFixture(t, root, "slot_a.json", "long_fall.json", 1)
	writeSaveFixture(t, root, "slot_b.json", "market.json", 2)
	writeRawFixture(t, root, "broken.json", []byte(`{"version":1}`))
	writeRawFixture(t, root, "notes.txt", []byte("ignore me"))

	slots, err := listSlotsInDir(root, 4)
	if err != nil {
		t.Fatalf("list slots: %v", err)
	}
	if len(slots) != 3 {
		t.Fatalf("expected 3 valid slots, got %d", len(slots))
	}

	if slots[0].FileName != "slot_a.json" {
		t.Fatalf("expected first slot to be slot_a.json, got %q", slots[0].FileName)
	}
	if slots[1].FileName != "slot_b.json" {
		t.Fatalf("expected second slot to be slot_b.json, got %q", slots[1].FileName)
	}
	if slots[2].FileName != "slot_c.json" {
		t.Fatalf("expected third slot to be slot_c.json, got %q", slots[2].FileName)
	}
	if slots[1].Snapshot == nil || slots[1].Snapshot.Level != "market.json" {
		t.Fatalf("expected slot_b snapshot to load market.json, got %+v", slots[1].Snapshot)
	}
}

func TestListSlotsInDirMissingDirectoryReturnsEmpty(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")

	slots, err := listSlotsInDir(root, 4)
	if err != nil {
		t.Fatalf("list slots from missing directory: %v", err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected no slots for missing directory, got %d", len(slots))
	}
}

func writeSaveFixture(t *testing.T, root, name, level string, gear int) {
	t.Helper()
	writeRawFixture(t, root, name, []byte("{\n  \"version\": 1,\n  \"level\": \""+level+"\",\n  \"player\": {\n    \"health\": {\"initial\": 5, \"current\": 3},\n    \"abilities\": {\"doubleJump\": true, \"wallGrab\": false, \"anchor\": true, \"heal\": false},\n    \"gearCount\": "+itoa(gear)+",\n    \"transform\": {\"x\": 0, \"y\": 0, \"scaleX\": 1, \"scaleY\": 1, \"rotation\": 0},\n    \"safeRespawn\": {\"x\": 0, \"y\": 0, \"initialized\": false},\n    \"facingLeft\": false\n  },\n  \"savedAt\": \""+time.Date(2026, time.April, gear, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)+"\"\n}\n"))
}

func writeRawFixture(t *testing.T, root, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func itoa(value int) string {
	return string(rune('0' + value))
}
