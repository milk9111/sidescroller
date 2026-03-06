package autotile

import "testing"

func TestDefaultMaskOrderMatchesExpected47Order(t *testing.T) {
	expected := []uint8{28, 124, 112, 16, 247, 223, 125, 31, 255, 241, 17, 253, 127, 95, 7, 199, 193, 1, 117, 87, 245, 4, 68, 64, 0, 213, 93, 215, 23, 209, 116, 92, 20, 84, 80, 29, 113, 197, 71, 21, 85, 81, 221, 119, 5, 69, 65}
	order := DefaultMaskOrder()
	if len(order) != len(expected) {
		t.Fatalf("expected %d masks, got %d", len(expected), len(order))
	}
	for index := range expected {
		if order[index] != expected[index] {
			t.Fatalf("expected mask %d at index %d, got %d", expected[index], index, order[index])
		}
	}
}

func TestDefaultMaskOrderHas47Variants(t *testing.T) {
	order := DefaultMaskOrder()
	if got := len(order); got != 47 {
		t.Fatalf("expected 47 autotile masks, got %d", got)
	}
	seen := make(map[uint8]struct{}, len(order))
	for _, mask := range order {
		if canonical := Canonicalize(mask); canonical != mask {
			t.Fatalf("mask %08b was not canonicalized", mask)
		}
		if _, exists := seen[mask]; exists {
			t.Fatalf("duplicate mask %08b", mask)
		}
		seen[mask] = struct{}{}
	}
}

func TestCanonicalizeDropsInvalidCorners(t *testing.T) {
	mask := BuildMask(false, true, false, true, true, false, false, false)
	if mask&MaskNorthWest != 0 {
		t.Fatalf("expected northwest corner bit to be cleared, got %08b", mask)
	}

	mask = BuildMask(true, false, false, true, true, false, false, false)
	if mask&MaskNorthWest == 0 {
		t.Fatalf("expected northwest corner to be preserved, got %08b", mask)
	}
}

func TestResolveOffsetUsesRemapWhenPresent(t *testing.T) {
	order := DefaultMaskOrder()
	remap := make([]int, len(order))
	for index := range remap {
		remap[index] = len(order) - index - 1
	}
	offset, ok := ResolveOffset(order[5], remap)
	if !ok {
		t.Fatal("expected remapped mask to resolve")
	}
	if offset != remap[5] {
		t.Fatalf("expected offset %d, got %d", remap[5], offset)
	}
}
