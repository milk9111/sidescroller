package entity

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/levels"
)

func TestLevelTileOccupiedTreatsZeroIndexTileWithUsageAsFilled(t *testing.T) {
	if !levelTileOccupied(0, &levels.TileInfo{Path: "terrain.png", Index: 0, TileW: 32, TileH: 32}) {
		t.Fatal("expected zero-index tile with usage metadata to count as occupied")
	}
	if levelTileOccupied(0, nil) {
		t.Fatal("expected nil usage with zero tile id to count as empty")
	}
}

func TestAddMergedTileCollidersTreatsZeroIndexTileWithUsageAsSolid(t *testing.T) {
	w := ecs.NewWorld()
	layer := []int{0}
	usage := []*levels.TileInfo{{Path: "terrain.png", Index: 0, TileW: 32, TileH: 32}}

	if err := addMergedTileColliders(w, layer, usage, 1, 1, 32); err != nil {
		t.Fatalf("addMergedTileColliders returned error: %v", err)
	}

	bodyCount := 0
	ecs.ForEach(w, component.PhysicsBodyComponent.Kind(), func(_ ecs.Entity, body *component.PhysicsBody) {
		if body != nil {
			bodyCount++
		}
	})
	if bodyCount != 1 {
		t.Fatalf("expected 1 collider for zero-index occupied tile, got %d", bodyCount)
	}
}
