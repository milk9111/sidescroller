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

func TestBuildLevelGridDataTracksOccupiedAndSolidCells(t *testing.T) {
	lvl := &levels.Level{
		Width:  2,
		Height: 2,
		Layers: [][]int{
			{0, 0, 0, 0},
			{0, 0, 1, 0},
		},
		TilesetUsage: [][]*levels.TileInfo{
			{{Path: "terrain.png", Index: 0, TileW: 32, TileH: 32}, nil, nil, nil},
			nil,
		},
		LayerMeta: []levels.LayerMeta{{Physics: true}, {Physics: false}},
	}

	grid := buildLevelGridData(lvl, 32)
	if grid == nil {
		t.Fatal("expected non-nil level grid")
	}
	if grid.Width != 2 || grid.Height != 2 {
		t.Fatalf("expected 2x2 grid, got %dx%d", grid.Width, grid.Height)
	}
	if !grid.CellOccupied(0, 0) {
		t.Fatal("expected (0,0) to be occupied from tileset usage")
	}
	if !grid.CellSolid(0, 0) {
		t.Fatal("expected (0,0) to be solid from physics layer")
	}
	if !grid.CellOccupied(0, 1) {
		t.Fatal("expected (0,1) to be occupied from non-zero tile id")
	}
	if grid.CellSolid(0, 1) {
		t.Fatal("expected (0,1) to remain non-solid because its layer is non-physics")
	}
	if grid.CellOccupied(1, 1) {
		t.Fatal("expected (1,1) to be empty")
	}
}
