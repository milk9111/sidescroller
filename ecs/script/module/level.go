package module

import (
	"fmt"
	"math"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func LevelModule() Module {
	return Module{
		Name: "level",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["current_cell"] = &tengo.UserFunction{Name: "current_cell", Value: func(args ...tengo.Object) (tengo.Object, error) {
				grid, err := levelGrid(world)
				if err != nil {
					return levelCellObject(nil, 0, 0), err
				}

				cellX, cellY, err := entityCellPosition(world, target, grid.TileSize)
				return levelCellObject(grid, cellX, cellY), err
			}}

			values["cell"] = &tengo.UserFunction{Name: "cell", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return levelCellObject(nil, 0, 0), fmt.Errorf("cell requires 2 arguments: x and y")
				}

				grid, err := levelGrid(world)
				if err != nil {
					return levelCellObject(nil, int(objectAsFloat(args[0])), int(objectAsFloat(args[1]))), err
				}

				cellX := int(objectAsFloat(args[0]))
				cellY := int(objectAsFloat(args[1]))
				return levelCellObject(grid, cellX, cellY), nil
			}}

			values["down_solid"] = &tengo.UserFunction{Name: "down_solid", Value: func(args ...tengo.Object) (tengo.Object, error) {
				grid, err := levelGrid(world)
				if err != nil {
					return tengo.FalseValue, err
				}

				hasSolid, err := entityDownSolid(world, target, grid)
				if err != nil {
					return tengo.FalseValue, err
				}

				return boolObject(hasSolid), nil
			}}

			values["snap_to_down_solid"] = &tengo.UserFunction{Name: "snap_to_down_solid", Value: func(args ...tengo.Object) (tengo.Object, error) {
				grid, err := levelGrid(world)
				if err != nil {
					return tengo.FalseValue, err
				}

				snapped, err := snapEntityToDownSolid(world, target, grid)
				if err != nil {
					return tengo.FalseValue, err
				}

				return boolObject(snapped), nil
			}}

			return values
		},
	}
}

type supportSample struct {
	x float64
	y float64
}

func entityDownSolid(world *ecs.World, target ecs.Entity, grid *component.LevelGrid) (bool, error) {
	_, samples, _, downX, downY, err := entityDownFace(world, target)
	if err != nil {
		return false, err
	}
	if grid == nil || len(samples) == 0 {
		return false, nil
	}

	probeEpsilon := math.Max(0.5, grid.TileSize*0.02)
	for _, sample := range samples {
		cellX := int(math.Floor((sample.x + downX*probeEpsilon) / grid.TileSize))
		cellY := int(math.Floor((sample.y + downY*probeEpsilon) / grid.TileSize))
		if grid.CellSolid(cellX, cellY) {
			return true, nil
		}
	}

	return false, nil
}

func snapEntityToDownSolid(world *ecs.World, target ecs.Entity, grid *component.LevelGrid) (bool, error) {
	transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return false, fmt.Errorf("entity does not have a transform component")
	}
	if grid == nil {
		return false, nil
	}

	faceDot, samples, rotation, downX, downY, err := entityDownFace(world, target)
	if err != nil {
		return false, err
	}
	if len(samples) == 0 {
		return false, nil
	}

	probeEpsilon := math.Max(0.5, grid.TileSize*0.02)
	stepX := 0
	stepY := 0
	if math.Abs(downX) > math.Abs(downY) {
		stepX = int(math.Copysign(1, downX))
	} else if math.Abs(downY) > 0 {
		stepY = int(math.Copysign(1, downY))
	}
	const maxProbeSteps = 1
	bestDelta := math.Inf(1)
	bestFound := false
	for _, sample := range samples {
		baseCellX := int(math.Floor((sample.x + downX*probeEpsilon) / grid.TileSize))
		baseCellY := int(math.Floor((sample.y + downY*probeEpsilon) / grid.TileSize))
		for step := 0; step <= maxProbeSteps; step++ {
			cellX := baseCellX + stepX*step
			cellY := baseCellY + stepY*step
			if !grid.CellSolid(cellX, cellY) {
				continue
			}

			targetFaceDot := cellBoundaryDot(cellX, cellY, grid.TileSize, downX, downY)
			delta := targetFaceDot - faceDot
			if !bestFound || math.Abs(delta) < math.Abs(bestDelta) {
				bestDelta = delta
				bestFound = true
			}
		}
	}

	if !bestFound {
		return false, nil
	}

	transform.X += downX * bestDelta
	transform.Y += downY * bestDelta
	transform.Rotation = rotation

	return true, nil
}

func entityDownFace(world *ecs.World, target ecs.Entity) (float64, []supportSample, float64, float64, float64, error) {
	transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return 0, nil, 0, 0, 0, fmt.Errorf("entity does not have a transform component")
	}

	tx, ty, width, height, rotation, err := levelEntityRect(world, target, transform)
	if err != nil {
		return 0, nil, 0, 0, 0, err
	}

	downX, downY := scriptDownVector(rotation)
	tangentX := downY
	tangentY := -downX

	localCorners := [4]supportSample{{0, 0}, {width, 0}, {0, height}, {width, height}}
	type projectedCorner struct {
		x     float64
		y     float64
		dot   float64
		along float64
	}
	projected := make([]projectedCorner, 0, len(localCorners))
	maxDot := math.Inf(-1)
	for _, corner := range localCorners {
		worldX := tx + corner.x*math.Cos(rotation) - corner.y*math.Sin(rotation)
		worldY := ty + corner.x*math.Sin(rotation) + corner.y*math.Cos(rotation)
		dot := worldX*downX + worldY*downY
		along := worldX*tangentX + worldY*tangentY
		projected = append(projected, projectedCorner{x: worldX, y: worldY, dot: dot, along: along})
		if dot > maxDot {
			maxDot = dot
		}
	}

	const epsilon = 0.001
	minAlong := math.Inf(1)
	maxAlong := math.Inf(-1)
	var start, end supportSample
	for _, corner := range projected {
		if math.Abs(corner.dot-maxDot) > epsilon {
			continue
		}
		if corner.along < minAlong {
			minAlong = corner.along
			start = supportSample{x: corner.x, y: corner.y}
		}
		if corner.along > maxAlong {
			maxAlong = corner.along
			end = supportSample{x: corner.x, y: corner.y}
		}
	}

	samples := []supportSample{
		{x: (start.x + end.x) * 0.5, y: (start.y + end.y) * 0.5},
		start,
		end,
	}

	return maxDot, samples, rotation, downX, downY, nil
}

func levelEntityRect(world *ecs.World, target ecs.Entity, transform *component.Transform) (x, y, width, height, rotation float64, err error) {
	if transform == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("entity does not have a transform component")
	}

	x = transform.X
	y = transform.Y
	scaleX := transform.ScaleX
	scaleY := transform.ScaleY
	if transform.Parent != 0 || transform.WorldX != 0 || transform.WorldY != 0 {
		x = transform.WorldX
		y = transform.WorldY
		scaleX = transform.WorldScaleX
		scaleY = transform.WorldScaleY
	}
	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}
	rotation = scriptRotationRadians(world, target, transform)

	width = 32 * math.Abs(scaleX)
	height = 32 * math.Abs(scaleY)
	if sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind()); ok && sprite != nil && sprite.Image != nil {
		imgW := sprite.Image.Bounds().Dx()
		imgH := sprite.Image.Bounds().Dy()
		if sprite.UseSource {
			imgW = sprite.Source.Dx()
			imgH = sprite.Source.Dy()
		}
		if imgW > 0 {
			width = float64(imgW) * math.Abs(scaleX)
		}
		if imgH > 0 {
			height = float64(imgH) * math.Abs(scaleY)
		}
	}

	return x, y, width, height, rotation, nil
}

func cellBoundaryDot(cellX, cellY int, tileSize, downX, downY float64) float64 {
	worldX := float64(cellX) * tileSize
	worldY := float64(cellY) * tileSize
	if math.Abs(downX) > math.Abs(downY) {
		if downX > 0 {
			return worldX
		}
		return -(worldX + tileSize)
	}
	if downY > 0 {
		return worldY
	}
	return -(worldY + tileSize)
}

func levelGrid(world *ecs.World) (*component.LevelGrid, error) {
	if world == nil {
		return nil, fmt.Errorf("world is nil")
	}

	ent, ok := ecs.First(world, component.LevelGridComponent.Kind())
	if !ok {
		return nil, fmt.Errorf("level grid component not found")
	}

	grid, ok := ecs.Get(world, ent, component.LevelGridComponent.Kind())
	if !ok || grid == nil {
		return nil, fmt.Errorf("level grid component not found")
	}

	return grid, nil
}

func entityCellPosition(world *ecs.World, target ecs.Entity, tileSize float64) (int, int, error) {
	transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return 0, 0, fmt.Errorf("entity does not have a transform component")
	}

	if tileSize <= 0 {
		tileSize = 32
	}

	x := transform.X
	y := transform.Y
	if transform.Parent != 0 || transform.WorldX != 0 || transform.WorldY != 0 {
		x = transform.WorldX
		y = transform.WorldY
	}

	return int(math.Floor(x / tileSize)), int(math.Floor(y / tileSize)), nil
}

func levelCellObject(grid *component.LevelGrid, cellX, cellY int) *tengo.ImmutableMap {
	tileSize := 0.0
	inBounds := false
	occupied := false
	solid := false
	index := -1

	if grid != nil {
		tileSize = grid.TileSize
		if tileSize <= 0 {
			tileSize = 32
		}
		inBounds = grid.InBounds(cellX, cellY)
		index = grid.CellIndex(cellX, cellY)
		occupied = grid.CellOccupied(cellX, cellY)
		solid = grid.CellSolid(cellX, cellY)
	}

	worldX := float64(cellX) * tileSize
	worldY := float64(cellY) * tileSize

	return &tengo.ImmutableMap{Value: map[string]tengo.Object{
		"x":         &tengo.Int{Value: int64(cellX)},
		"y":         &tengo.Int{Value: int64(cellY)},
		"index":     &tengo.Int{Value: int64(index)},
		"tile_size": &tengo.Float{Value: tileSize},
		"world_x":   &tengo.Float{Value: worldX},
		"world_y":   &tengo.Float{Value: worldY},
		"center_x":  &tengo.Float{Value: worldX + tileSize/2},
		"center_y":  &tengo.Float{Value: worldY + tileSize/2},
		"in_bounds": boolObject(inBounds),
		"occupied":  boolObject(occupied),
		"solid":     boolObject(solid),
	}}
}

func boolObject(value bool) tengo.Object {
	if value {
		return tengo.TrueValue
	}
	return tengo.FalseValue
}
