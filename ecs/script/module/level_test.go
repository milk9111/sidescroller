package module

import (
	"math"
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func makeLevelGrid(width, height int, solids [][2]int) *component.LevelGrid {
	grid := &component.LevelGrid{
		Width:    width,
		Height:   height,
		TileSize: 32,
		Occupied: make([]bool, width*height),
		Solid:    make([]bool, width*height),
	}
	for _, cell := range solids {
		idx := cell[1]*width + cell[0]
		if idx < 0 || idx >= len(grid.Solid) {
			continue
		}
		grid.Occupied[idx] = true
		grid.Solid[idx] = true
	}
	return grid
}

func TestLevelModuleCurrentCellAndNeighborLookup(t *testing.T) {
	w := ecs.NewWorld()
	levelEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, levelEntity, component.LevelGridComponent.Kind(), &component.LevelGrid{
		Width:    4,
		Height:   3,
		TileSize: 32,
		Occupied: []bool{
			false, true, false, false,
			false, false, true, false,
			false, false, false, false,
		},
		Solid: []bool{
			false, false, false, false,
			false, false, true, false,
			false, false, false, false,
		},
	}); err != nil {
		t.Fatalf("add level grid: %v", err)
	}

	entity := ecs.CreateEntity(w)
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), &component.Transform{X: 48, Y: 40}); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	mod := LevelModule().Build(w, nil, entity, entity)
	currentObj, err := mod["current_cell"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("current_cell returned error: %v", err)
	}
	current := currentObj.(*tengo.ImmutableMap)
	assertIntField(t, current, "x", 1)
	assertIntField(t, current, "y", 1)
	assertBoolField(t, current, "in_bounds", true)
	assertBoolField(t, current, "occupied", false)

	neighborObj, err := mod["cell"].(*tengo.UserFunction).Value(&tengo.Int{Value: 2}, &tengo.Int{Value: 1})
	if err != nil {
		t.Fatalf("cell returned error: %v", err)
	}
	neighbor := neighborObj.(*tengo.ImmutableMap)
	assertIntField(t, neighbor, "index", 6)
	assertBoolField(t, neighbor, "occupied", true)
	assertBoolField(t, neighbor, "solid", true)

	oobObj, err := mod["cell"].(*tengo.UserFunction).Value(&tengo.Int{Value: 10}, &tengo.Int{Value: 1})
	if err != nil {
		t.Fatalf("cell out-of-bounds returned error: %v", err)
	}
	oob := oobObj.(*tengo.ImmutableMap)
	assertBoolField(t, oob, "in_bounds", false)
	assertIntField(t, oob, "index", -1)
}

func TestLevelModuleForwardSolidUsesEntityRotation(t *testing.T) {
	w := ecs.NewWorld()
	levelEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, levelEntity, component.LevelGridComponent.Kind(), makeLevelGrid(4, 4, [][2]int{{2, 1}, {1, 2}})); err != nil {
		t.Fatalf("add level grid: %v", err)
	}

	entity := ecs.CreateEntity(w)
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), &component.Transform{X: 32, Y: 32, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	mod := LevelModule().Build(w, nil, entity, entity)
	forwardObj, err := mod["forward_solid"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("forward_solid returned error: %v", err)
	}
	if forwardObj != tengo.TrueValue {
		t.Fatal("expected forward_solid to detect the cell ahead at rotation 0")
	}

	transform, _ := ecs.Get(w, entity, component.TransformComponent.Kind())
	transform.Rotation = math.Pi / 2

	forwardObj, err = mod["forward_solid"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("forward_solid after rotation returned error: %v", err)
	}
	if forwardObj != tengo.TrueValue {
		t.Fatal("expected forward_solid to detect the cell ahead at rotation 90 degrees")
	}

	transform.Rotation = math.Pi
	forwardObj, err = mod["forward_solid"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("forward_solid after second rotation returned error: %v", err)
	}
	if forwardObj != tengo.FalseValue {
		t.Fatal("expected forward_solid to be false with no solid cell directly ahead")
	}
}

func TestLevelModuleForwardSolidIgnoresDiagonalBelowRightTile(t *testing.T) {
	w := ecs.NewWorld()
	levelEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, levelEntity, component.LevelGridComponent.Kind(), makeLevelGrid(4, 4, [][2]int{{2, 2}})); err != nil {
		t.Fatalf("add level grid: %v", err)
	}

	entity := ecs.CreateEntity(w)
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), &component.Transform{X: 32, Y: 32, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	mod := LevelModule().Build(w, nil, entity, entity)
	forwardObj, err := mod["forward_solid"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("forward_solid returned error: %v", err)
	}
	if forwardObj != tengo.FalseValue {
		t.Fatal("expected forward_solid to ignore a tile that is only diagonally below-right")
	}
}

func TestSnapEntityToDownSolidReattachesAfterTopLeftCornerRotation(t *testing.T) {
	w := ecs.NewWorld()
	levelEntity := ecs.CreateEntity(w)
	grid := makeLevelGrid(10, 10, [][2]int{
		{4, 3}, {5, 3}, {6, 3},
		{3, 4}, {4, 4}, {5, 4}, {6, 4},
		{3, 5}, {4, 5}, {5, 5}, {6, 5},
		{3, 6}, {4, 6}, {5, 6}, {6, 6},
	})
	if err := ecs.Add(w, levelEntity, component.LevelGridComponent.Kind(), grid); err != nil {
		t.Fatalf("add level grid: %v", err)
	}

	entity := ecs.CreateEntity(w)
	transform := &component.Transform{X: 96, Y: 56, ScaleX: 1, ScaleY: 1}
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), transform); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	hasSolid, err := entityDownSolid(w, entity, grid)
	if err != nil {
		t.Fatalf("entityDownSolid returned error: %v", err)
	}
	if hasSolid {
		t.Fatal("expected rotated corner pose to start detached from support")
	}

	snapped, err := snapEntityToDownSolid(w, entity, grid)
	if err != nil {
		t.Fatalf("snapEntityToDownSolid returned error: %v", err)
	}
	if !snapped {
		t.Fatal("expected snapEntityToDownSolid to reattach to the top-left corner surface")
	}
	if transform.X != 96 || transform.Y != 64 {
		t.Fatalf("expected snapped transform to be at (96,64), got (%v,%v)", transform.X, transform.Y)
	}

	hasSolid, err = entityDownSolid(w, entity, grid)
	if err != nil {
		t.Fatalf("entityDownSolid after snap returned error: %v", err)
	}
	if !hasSolid {
		t.Fatal("expected snapped entity to detect supporting solid")
	}
}

func TestSnapEntityToDownSolidSkipsFullTileExtendedSnap(t *testing.T) {
	w := ecs.NewWorld()
	levelEntity := ecs.CreateEntity(w)
	grid := makeLevelGrid(10, 10, [][2]int{{4, 3}})
	if err := ecs.Add(w, levelEntity, component.LevelGridComponent.Kind(), grid); err != nil {
		t.Fatalf("add level grid: %v", err)
	}

	entity := ecs.CreateEntity(w)
	transform := &component.Transform{X: 96, Y: 32, ScaleX: 1, ScaleY: 1}
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), transform); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	snapped, err := snapEntityToDownSolid(w, entity, grid)
	if err != nil {
		t.Fatalf("snapEntityToDownSolid returned error: %v", err)
	}
	if snapped {
		t.Fatal("expected snapEntityToDownSolid to reject a full-tile extended snap")
	}
	if transform.X != 96 || transform.Y != 32 {
		t.Fatalf("expected transform to remain at (96,32), got (%v,%v)", transform.X, transform.Y)
	}
}

func assertIntField(t *testing.T, m *tengo.ImmutableMap, key string, want int64) {
	t.Helper()
	obj, ok := m.Value[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	value, ok := obj.(*tengo.Int)
	if !ok {
		t.Fatalf("field %q was %T, want *tengo.Int", key, obj)
	}
	if value.Value != want {
		t.Fatalf("field %q = %d, want %d", key, value.Value, want)
	}
}

func assertBoolField(t *testing.T, m *tengo.ImmutableMap, key string, want bool) {
	t.Helper()
	obj, ok := m.Value[key]
	if !ok {
		t.Fatalf("missing field %q", key)
	}
	got := obj == tengo.TrueValue
	if got != want {
		t.Fatalf("field %q = %t, want %t", key, got, want)
	}
}
