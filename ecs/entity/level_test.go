package entity

import (
	"math"
	"sort"
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/levels"
)

func TestBuildMergedSpikeHazardsMergesHorizontalRunsWithTrimmedEnds(t *testing.T) {
	hazards := buildMergedSpikeHazards([]loadedSpikePlacement{
		{x: 0, y: 0, rotation: 0, layerIndex: 2},
		{x: 32, y: 0, rotation: 0, layerIndex: 2},
		{x: 64, y: 0, rotation: 0, layerIndex: 2},
	})

	if len(hazards) != 1 {
		t.Fatalf("expected 1 merged hazard, got %d", len(hazards))
	}

	hazard := hazards[0]
	assertFloatApprox(t, hazard.x, 4)
	assertFloatApprox(t, hazard.y, 6)
	assertFloatApprox(t, hazard.w, 88)
	assertFloatApprox(t, hazard.h, 26)
	if hazard.layerIndex != 2 {
		t.Fatalf("expected layer 2, got %d", hazard.layerIndex)
	}
}

func TestBuildMergedSpikeHazardsMergesVerticalRunsWithTrimmedEnds(t *testing.T) {
	hazards := buildMergedSpikeHazards([]loadedSpikePlacement{
		{x: 0, y: 0, rotation: 90, layerIndex: 0},
		{x: 0, y: 32, rotation: 90, layerIndex: 0},
	})

	if len(hazards) != 1 {
		t.Fatalf("expected 1 merged hazard, got %d", len(hazards))
	}

	hazard := hazards[0]
	assertFloatApprox(t, hazard.x, 0)
	assertFloatApprox(t, hazard.y, 4)
	assertFloatApprox(t, hazard.w, 26)
	assertFloatApprox(t, hazard.h, 56)
}

func TestBuildMergedSpikeHazardsKeepsDistinctRowsLayersAndSingles(t *testing.T) {
	hazards := buildMergedSpikeHazards([]loadedSpikePlacement{
		{x: 0, y: 0, rotation: 0, layerIndex: 0},
		{x: 32, y: 0, rotation: 0, layerIndex: 1},
		{x: 0, y: 32, rotation: 0, layerIndex: 0},
	})

	if len(hazards) != 3 {
		t.Fatalf("expected 3 separate hazards, got %d", len(hazards))
	}

	sort.Slice(hazards, func(i, j int) bool {
		if hazards[i].layerIndex != hazards[j].layerIndex {
			return hazards[i].layerIndex < hazards[j].layerIndex
		}
		if hazards[i].y != hazards[j].y {
			return hazards[i].y < hazards[j].y
		}
		return hazards[i].x < hazards[j].x
	})

	assertFloatApprox(t, hazards[0].x, 0)
	assertFloatApprox(t, hazards[0].y, 6)
	assertFloatApprox(t, hazards[1].x, 0)
	assertFloatApprox(t, hazards[1].y, 38)
	assertFloatApprox(t, hazards[2].x, 32)
	assertFloatApprox(t, hazards[2].y, 6)
	for i, hazard := range hazards {
		assertFloatApprox(t, hazard.w, 32)
		assertFloatApprox(t, hazard.h, 26)
		if i < 2 && hazard.layerIndex != 0 {
			t.Fatalf("expected hazards[ %d ] on layer 0, got %d", i, hazard.layerIndex)
		}
	}
	if hazards[2].layerIndex != 1 {
		t.Fatalf("expected third hazard on layer 1, got %d", hazards[2].layerIndex)
	}
}

func assertFloatApprox(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

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
