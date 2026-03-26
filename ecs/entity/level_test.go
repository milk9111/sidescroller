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

	if err := addMergedTileColliders(w, 0, layer, usage, 1, 1, 32); err != nil {
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

func TestRebuildMergedLevelPhysicsMergesAcrossPhysicsLayers(t *testing.T) {
	w := ecs.NewWorld()
	lvl := &levels.Level{
		Width:  2,
		Height: 1,
		Layers: [][]int{
			{1, 0},
			{0, 1},
		},
		LayerMeta: []levels.LayerMeta{{Physics: true}, {Physics: true}},
	}

	if err := RebuildMergedLevelPhysics(w, lvl, 32); err != nil {
		t.Fatalf("RebuildMergedLevelPhysics() error = %v", err)
	}

	bodyCount := 0
	ecs.ForEach2(w, component.MergedLevelPhysicsComponent.Kind(), component.PhysicsBodyComponent.Kind(), func(_ ecs.Entity, _ *component.MergedLevelPhysics, body *component.PhysicsBody) {
		bodyCount++
		if body.Width != 64 || body.Height != 32 {
			t.Fatalf("expected merged collider size 64x32, got %vx%v", body.Width, body.Height)
		}
	})
	if bodyCount != 1 {
		t.Fatalf("expected 1 merged collider across layers, got %d", bodyCount)
	}
}

func TestRebuildMergedLevelPhysicsAssignsWorldCollisionLayer(t *testing.T) {
	w := ecs.NewWorld()
	lvl := &levels.Level{
		Width:     1,
		Height:    1,
		Layers:    [][]int{{1}},
		LayerMeta: []levels.LayerMeta{{Physics: true}},
	}

	if err := RebuildMergedLevelPhysics(w, lvl, 32); err != nil {
		t.Fatalf("RebuildMergedLevelPhysics() error = %v", err)
	}

	count := 0
	ecs.ForEach2(w, component.MergedLevelPhysicsComponent.Kind(), component.CollisionLayerComponent.Kind(), func(_ ecs.Entity, _ *component.MergedLevelPhysics, layer *component.CollisionLayer) {
		count++
		if layer.Category != component.CollisionCategoryWorld {
			t.Fatalf("expected merged level physics category %d, got %d", component.CollisionCategoryWorld, layer.Category)
		}
		if layer.Mask != ^uint32(0) {
			t.Fatalf("expected merged level physics mask all-bits set, got %d", layer.Mask)
		}
	})

	if count != 1 {
		t.Fatalf("expected 1 merged collider collision layer, got %d", count)
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

func TestBuildLevelGridDataIgnoresInactiveLayers(t *testing.T) {
	inactive := false
	lvl := &levels.Level{
		Width:     1,
		Height:    1,
		Layers:    [][]int{{1}},
		LayerMeta: []levels.LayerMeta{{Physics: true, Active: &inactive}},
	}

	grid := buildLevelGridData(lvl, 32)
	if grid.CellOccupied(0, 0) {
		t.Fatal("expected inactive layer tile to be ignored for occupancy")
	}
	if grid.CellSolid(0, 0) {
		t.Fatal("expected inactive layer tile to be ignored for solidity")
	}
}

func TestLoadLevelToWorldSkipsInactiveLayerEntitiesAndPhysics(t *testing.T) {
	inactive := false
	w := ecs.NewWorld()
	lvl := &levels.Level{
		Width:     1,
		Height:    1,
		Layers:    [][]int{{1}},
		LayerMeta: []levels.LayerMeta{{Physics: true, Active: &inactive}},
		Entities:  []levels.Entity{{Type: "enemy", Props: map[string]interface{}{"prefab": "enemy.yaml", "layer": 0}}},
	}

	if err := LoadLevelToWorld(w, lvl); err != nil {
		t.Fatalf("LoadLevelToWorld() error = %v", err)
	}

	staticTiles := 0
	physicsBodies := 0
	gameEntities := 0
	ecs.ForEach(w, component.StaticTileComponent.Kind(), func(_ ecs.Entity, _ *component.StaticTile) {
		staticTiles++
	})
	ecs.ForEach(w, component.PhysicsBodyComponent.Kind(), func(_ ecs.Entity, _ *component.PhysicsBody) {
		physicsBodies++
	})
	ecs.ForEach(w, component.GameEntityIDComponent.Kind(), func(_ ecs.Entity, _ *component.GameEntityID) {
		gameEntities++
	})
	if staticTiles != 0 {
		t.Fatalf("expected no static tiles from inactive layer, got %d", staticTiles)
	}
	if physicsBodies != 0 {
		t.Fatalf("expected no colliders from inactive layer, got %d", physicsBodies)
	}
	if gameEntities != 0 {
		t.Fatalf("expected no level entities from inactive layer, got %d", gameEntities)
	}
}

func TestLoadLevelToWorldConfiguresTriggerEntity(t *testing.T) {
	w := ecs.NewWorld()
	lvl := &levels.Level{
		Width:     1,
		Height:    1,
		Layers:    [][]int{{0}},
		LayerMeta: []levels.LayerMeta{{Physics: false}},
		Entities: []levels.Entity{{
			ID:   "trigger_1",
			Type: "trigger",
			X:    96,
			Y:    128,
			Props: map[string]interface{}{
				"layer": 0,
				"w":     64,
				"h":     48,
				"components": map[string]interface{}{
					"script": map[string]interface{}{
						"path": "triggers/disposal_8.tengo",
					},
				},
			},
		}},
	}

	if err := LoadLevelToWorld(w, lvl); err != nil {
		t.Fatalf("LoadLevelToWorld() error = %v", err)
	}

	count := 0
	ecs.ForEach3(w, component.TriggerComponent.Kind(), component.TransformComponent.Kind(), component.ScriptComponent.Kind(), func(entity ecs.Entity, trigger *component.Trigger, transform *component.Transform, script *component.Script) {
		count++
		if trigger == nil || transform == nil || script == nil {
			t.Fatal("expected trigger, transform, and script components")
		}
		if trigger.Name != "trigger_1" {
			t.Fatalf("expected trigger name trigger_1, got %q", trigger.Name)
		}
		if trigger.Bounds.W != 64 || trigger.Bounds.H != 48 {
			t.Fatalf("expected trigger bounds 64x48, got %vx%v", trigger.Bounds.W, trigger.Bounds.H)
		}
		if areaBounds, ok := ecs.Get(w, entity, component.AreaBoundsComponent.Kind()); ok && areaBounds != nil {
			if areaBounds.Bounds.W != 64 || areaBounds.Bounds.H != 48 {
				t.Fatalf("expected shared area bounds 64x48, got %vx%v", areaBounds.Bounds.W, areaBounds.Bounds.H)
			}
		} else {
			t.Fatal("expected shared area bounds on trigger entity")
		}
		if transform.X != 96 || transform.Y != 128 {
			t.Fatalf("expected trigger transform at (96,128), got (%v,%v)", transform.X, transform.Y)
		}
		if script.Path != "triggers/disposal_8.tengo" {
			t.Fatalf("expected trigger script path override, got %q", script.Path)
		}
	})
	if count != 1 {
		t.Fatalf("expected 1 trigger entity, got %d", count)
	}
}

func TestLoadLevelToWorldExpandsBreakableWallAreaIntoVisualTiles(t *testing.T) {
	w := ecs.NewWorld()
	lvl := &levels.Level{
		Width:  3,
		Height: 2,
		Layers: [][]int{{1, 0, 1, 1, 1, 1}},
		TilesetUsage: [][]*levels.TileInfo{{
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			nil,
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
		}},
		LayerMeta: []levels.LayerMeta{{Physics: true}},
		Entities: []levels.Entity{{
			ID:   "wall_1",
			Type: "breakable_wall",
			X:    32,
			Y:    0,
			Props: map[string]interface{}{
				"layer":  0,
				"w":      64,
				"h":      32,
				"prefab": "breakable_cracks.yaml",
			},
		}},
	}

	if err := LoadLevelToWorld(w, lvl); err != nil {
		t.Fatalf("LoadLevelToWorld() error = %v", err)
	}

	rootCount := 0
	physicsWidth := 0.0
	hurtboxWidth := 0.0
	rotationOffset := 0.0
	ecs.ForEach4(w, component.TransformComponent.Kind(), component.SpriteComponent.Kind(), component.EntityLayerComponent.Kind(), component.RenderLayerComponent.Kind(), func(e ecs.Entity, transform *component.Transform, sprite *component.Sprite, layer *component.EntityLayer, renderLayer *component.RenderLayer) {
		if layer == nil || layer.Index != 0 || transform == nil || sprite == nil || renderLayer == nil {
			return
		}
		if gameID, ok := ecs.Get(w, e, component.GameEntityIDComponent.Kind()); ok && gameID != nil {
			rootCount++
			if sprite.Disabled {
				t.Fatal("expected logical breakable wall root sprite to remain enabled for shared tile stamping")
			}
			if body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); ok && body != nil {
				physicsWidth = body.Width
			}
			if hurtboxes, ok := ecs.Get(w, e, component.HurtboxComponent.Kind()); ok && hurtboxes != nil && len(*hurtboxes) == 1 {
				hurtboxWidth = (*hurtboxes)[0].Width
			}
			if areaBounds, ok := ecs.Get(w, e, component.AreaBoundsComponent.Kind()); ok && areaBounds != nil {
				if areaBounds.Bounds.W != 64 || areaBounds.Bounds.H != 32 {
					t.Fatalf("expected area bounds 64x32, got %vx%v", areaBounds.Bounds.W, areaBounds.Bounds.H)
				}
			} else {
				t.Fatal("expected shared area bounds component on breakable wall")
			}
			if stamp, ok := ecs.Get(w, e, component.AreaTileStampComponent.Kind()); ok && stamp != nil {
				rotationOffset = stamp.RotationOffset
			} else {
				t.Fatal("expected shared area tile stamp component on breakable wall")
			}
			return
		}
	})
	if rootCount != 1 {
		t.Fatalf("expected 1 logical breakable wall root, got %d", rootCount)
	}
	if physicsWidth != 64 {
		t.Fatalf("expected resized breakable wall physics width 64, got %v", physicsWidth)
	}
	if hurtboxWidth != 64 {
		t.Fatalf("expected resized breakable wall hurtbox width 64, got %v", hurtboxWidth)
	}
	if rotationOffset != 0 {
		t.Fatalf("expected default breakable wall visual rotation offset 0, got %v", rotationOffset)
	}
}

func TestLoadLevelToWorldAppliesGenericAreaPrefabBoundsAndPhysics(t *testing.T) {
	w := ecs.NewWorld()
	lvl := &levels.Level{
		Width:  4,
		Height: 2,
		Layers: [][]int{{1, 1, 1, 1, 1, 1, 1, 1}},
		TilesetUsage: [][]*levels.TileInfo{{
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
			{Path: "cableways_tile.png", Index: 0, TileW: 32, TileH: 32},
		}},
		LayerMeta: []levels.LayerMeta{{Physics: true}},
		Entities: []levels.Entity{{
			ID:   "platform_1",
			Type: "solid_tile_platform",
			X:    64,
			Y:    32,
			Props: map[string]interface{}{
				"layer":  0,
				"w":      96,
				"h":      32,
				"prefab": "solid_tile_platform.yaml",
			},
		}},
	}

	if err := LoadLevelToWorld(w, lvl); err != nil {
		t.Fatalf("LoadLevelToWorld() error = %v", err)
	}

	rootCount := 0
	physicsWidth := 0.0
	physicsHeight := 0.0
	rotationOffset := 0.0
	overdraw := 0.0
	overdrawMode := component.AreaTileStampOverdrawMode("")
	playerFacingSide := component.AreaTileStampSide("")
	transformX := 0.0
	transformY := 0.0
	ecs.ForEach4(w, component.TransformComponent.Kind(), component.SpriteComponent.Kind(), component.EntityLayerComponent.Kind(), component.RenderLayerComponent.Kind(), func(e ecs.Entity, transform *component.Transform, sprite *component.Sprite, layer *component.EntityLayer, renderLayer *component.RenderLayer) {
		if layer == nil || layer.Index != 0 || transform == nil || sprite == nil || renderLayer == nil {
			return
		}
		if gameID, ok := ecs.Get(w, e, component.GameEntityIDComponent.Kind()); ok && gameID != nil {
			rootCount++
			transformX = transform.X
			transformY = transform.Y
			if sprite.Disabled {
				t.Fatal("expected generic solid tile platform root sprite to remain enabled for area tile stamping")
			}
			if body, ok := ecs.Get(w, e, component.PhysicsBodyComponent.Kind()); ok && body != nil {
				physicsWidth = body.Width
				physicsHeight = body.Height
			}
			if areaBounds, ok := ecs.Get(w, e, component.AreaBoundsComponent.Kind()); ok && areaBounds != nil {
				if areaBounds.Bounds.W != 96 || areaBounds.Bounds.H != 32 {
					t.Fatalf("expected generic area bounds 96x32, got %vx%v", areaBounds.Bounds.W, areaBounds.Bounds.H)
				}
			} else {
				t.Fatal("expected shared area bounds component on generic area prefab")
			}
			if stamp, ok := ecs.Get(w, e, component.AreaTileStampComponent.Kind()); ok && stamp != nil {
				rotationOffset = stamp.RotationOffset
				overdraw = stamp.Overdraw
				overdrawMode = stamp.OverdrawMode
				playerFacingSide = stamp.PlayerFacingSide
			} else {
				t.Fatal("expected shared area tile stamp component on generic area prefab")
			}
		}
	})
	if rootCount != 1 {
		t.Fatalf("expected 1 logical generic area prefab root, got %d", rootCount)
	}
	if transformX != 64 || transformY != 32 {
		t.Fatalf("expected generic area prefab transform at (64,32), got (%v,%v)", transformX, transformY)
	}
	if physicsWidth != 96 || physicsHeight != 32 {
		t.Fatalf("expected resized generic area prefab physics 96x32, got %vx%v", physicsWidth, physicsHeight)
	}
	if rotationOffset != 0 {
		t.Fatalf("expected default generic area prefab visual rotation offset 0, got %v", rotationOffset)
	}
	if overdraw != 6 || overdrawMode != component.AreaTileStampOverdrawNonPlayerFacing || playerFacingSide != component.AreaTileStampSideTop {
		t.Fatalf("expected solid tile platform overdraw 6 with non_player_facing/top mode, got (%v,%v,%v)", overdraw, overdrawMode, playerFacingSide)
	}
}
