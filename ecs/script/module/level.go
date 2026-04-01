package module

import (
	"fmt"
	"math"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/levels"
)

func LevelModule() Module {
	return Module{
		Name: "level",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["activate"] = &tengo.UserFunction{Name: "activate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("activate requires 1 argument: layer name")
				}
				layerName, err := layerNameArg(args[0])
				if err != nil {
					return tengo.FalseValue, err
				}
				if err := setLevelLayerActive(world, layerName, true); err != nil {
					return tengo.FalseValue, err
				}
				return tengo.TrueValue, nil
			}}

			values["deactivate"] = &tengo.UserFunction{Name: "deactivate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("deactivate requires 1 argument: layer name")
				}
				layerName, err := layerNameArg(args[0])
				if err != nil {
					return tengo.FalseValue, err
				}
				if err := setLevelLayerActive(world, layerName, false); err != nil {
					return tengo.FalseValue, err
				}
				return tengo.TrueValue, nil
			}}

			values["add_fade_out"] = &tengo.UserFunction{Name: "add_fade_out", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("add_fade_out requires 2 arguments: layer name and duration in frames")
				}
				layerName, err := layerNameArg(args[0])
				if err != nil {
					return tengo.FalseValue, err
				}
				duration := objectAsInt(args[1])
				if duration < 0 {
					return tengo.FalseValue, fmt.Errorf("duration must be non-negative")
				}
				if err := addLevelLayerFadeOut(world, layerName, duration); err != nil {
					return tengo.FalseValue, err
				}
				return tengo.TrueValue, nil
			}}

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

			values["forward_solid"] = &tengo.UserFunction{Name: "forward_solid", Value: func(args ...tengo.Object) (tengo.Object, error) {
				grid, err := levelGrid(world)
				if err != nil {
					return tengo.FalseValue, err
				}

				hasSolid, err := entityForwardSolid(world, target, grid)
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
	return faceSamplesSolid(grid, samples, downX, downY), nil
}

func entityForwardSolid(world *ecs.World, target ecs.Entity, grid *component.LevelGrid) (bool, error) {
	if grid == nil {
		return false, nil
	}

	transform, ok := ecs.Get(world, target, component.TransformComponent.Kind())
	if !ok || transform == nil {
		return false, fmt.Errorf("entity does not have a transform component")
	}

	rotation := scriptRotationRadians(world, target, transform)
	forwardX, forwardY := scriptForwardVector(rotation)
	cellX, cellY, err := entityCellPosition(world, target, grid.TileSize)
	if err != nil {
		return false, err
	}

	stepX, stepY := snappedDirectionStep(forwardX, forwardY)
	if stepX == 0 && stepY == 0 {
		return false, nil
	}

	return grid.CellSolid(cellX+stepX, cellY+stepY), nil
}

func faceSamplesSolid(grid *component.LevelGrid, samples []supportSample, dirX, dirY float64) bool {
	if grid == nil || len(samples) == 0 {
		return false
	}

	probeEpsilon := math.Max(0.5, grid.TileSize*0.02)
	for _, sample := range samples {
		cellX := int(math.Floor((sample.x + dirX*probeEpsilon) / grid.TileSize))
		cellY := int(math.Floor((sample.y + dirY*probeEpsilon) / grid.TileSize))
		if grid.CellSolid(cellX, cellY) {
			return true
		}
	}

	return false
}

func snappedDirectionStep(dirX, dirY float64) (int, int) {
	stepX := 0
	stepY := 0
	if math.Abs(dirX) >= math.Abs(dirY) && math.Abs(dirX) > 0 {
		stepX = int(math.Round(dirX))
	} else if math.Abs(dirY) > 0 {
		stepY = int(math.Round(dirY))
	}
	return stepX, stepY
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
	maxExtendedSnapDistance := grid.TileSize * 0.5
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
			if step > 0 && math.Abs(delta) > maxExtendedSnapDistance {
				continue
			}
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
	faceDot, samples, faceRotation, faceX, faceY := entitySupportFace(tx, ty, width, height, rotation, downX, downY)
	return faceDot, samples, faceRotation, faceX, faceY, nil
}

func entitySupportFace(tx, ty, width, height, rotation, dirX, dirY float64) (float64, []supportSample, float64, float64, float64) {
	tangentX := dirY
	tangentY := -dirX

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
		dot := worldX*dirX + worldY*dirY
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

	return maxDot, samples, rotation, dirX, dirY
}

func levelEntityRect(world *ecs.World, target ecs.Entity, transform *component.Transform) (x, y, width, height, rotation float64, err error) {
	if transform == nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("entity does not have a transform component")
	}

	x = transform.X
	y = transform.Y
	scaleX := transform.ScaleX
	scaleY := transform.ScaleY
	if transform.Parent != 0 {
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

func setLevelLayerActive(world *ecs.World, layerName string, active bool) error {
	runtimeComp, err := levelRuntime(world)
	if err != nil {
		return err
	}
	if runtimeComp.Level == nil {
		return fmt.Errorf("level runtime data not found")
	}

	layerIndex := findLevelLayerIndex(runtimeComp.Level, layerName)
	if layerIndex < 0 {
		return fmt.Errorf("level layer %q not found", layerName)
	}

	ensureLoadedLayerCapacity(runtimeComp, len(runtimeComp.Level.Layers))
	ensureLayerMeta(runtimeComp.Level, layerIndex)
	activeValue := active
	runtimeComp.Level.LayerMeta[layerIndex].Active = &activeValue

	if active && !runtimeComp.LoadedLayers[layerIndex] {
		tileSize := runtimeComp.TileSize
		if tileSize <= 0 {
			tileSize = 32
		}
		if err := levelentity.LoadLevelLayerToWorld(world, runtimeComp.Level, layerIndex, tileSize); err != nil {
			return err
		}
		runtimeComp.LoadedLayers[layerIndex] = true
	}

	setRuntimeLayerEntityState(world, layerIndex, active)
	if levelentityLayerHasPhysics(runtimeComp.Level, layerIndex) {
		tileSize := runtimeComp.TileSize
		if tileSize <= 0 {
			tileSize = 32
		}
		if err := levelentity.RebuildMergedLevelPhysics(world, runtimeComp.Level, tileSize); err != nil {
			return err
		}
	}
	if err := rebuildLevelGrid(world, runtimeComp); err != nil {
		return err
	}

	recordLevelLayerState(world, runtimeComp.Name, layerName, active)
	return nil
}

func recordLevelLayerState(world *ecs.World, levelName, layerName string, active bool) {
	if world == nil {
		return
	}
	levelName = strings.TrimSpace(levelName)
	layerName = strings.TrimSpace(layerName)
	if levelName == "" || layerName == "" {
		return
	}

	player, ok := ecs.First(world, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	stateMap, ok := ecs.Get(world, player, component.LevelLayerStateMapComponent.Kind())
	if !ok || stateMap == nil {
		stateMap = &component.LevelLayerStateMap{States: map[string]bool{}}
		_ = ecs.Add(world, player, component.LevelLayerStateMapComponent.Kind(), stateMap)
	}
	if stateMap.States == nil {
		stateMap.States = map[string]bool{}
	}

	stateMap.States[levelName+"#"+layerName] = active
}

func addLevelLayerFadeOut(world *ecs.World, layerName string, duration int) error {
	runtimeComp, err := levelRuntime(world)
	if err != nil {
		return err
	}
	if runtimeComp.Level == nil {
		return fmt.Errorf("level runtime data not found")
	}

	layerIndex := findLevelLayerIndex(runtimeComp.Level, layerName)
	if layerIndex < 0 {
		return fmt.Errorf("level layer %q not found", layerName)
	}

	ecs.ForEach2(world, component.EntityLayerComponent.Kind(), component.SpriteComponent.Kind(), func(e ecs.Entity, layer *component.EntityLayer, _ *component.Sprite) {
		if layer == nil || layer.Index != layerIndex {
			return
		}
		if fade, ok := ecs.Get(world, e, component.SpriteFadeOutComponent.Kind()); ok && fade != nil {
			fade.Frames = duration
			fade.TotalFrames = duration
			fade.Alpha = 1
			return
		}
		_ = ecs.Add(world, e, component.SpriteFadeOutComponent.Kind(), &component.SpriteFadeOut{
			Frames:      duration,
			TotalFrames: duration,
			Alpha:       1,
		})
	})

	return nil
}

func levelRuntime(world *ecs.World) (*component.LevelRuntime, error) {
	if world == nil {
		return nil, fmt.Errorf("world is nil")
	}

	ent, ok := ecs.First(world, component.LevelRuntimeComponent.Kind())
	if !ok {
		return nil, fmt.Errorf("level runtime component not found")
	}

	runtimeComp, ok := ecs.Get(world, ent, component.LevelRuntimeComponent.Kind())
	if !ok || runtimeComp == nil {
		return nil, fmt.Errorf("level runtime component not found")
	}

	return runtimeComp, nil
}

func ensureLoadedLayerCapacity(runtimeComp *component.LevelRuntime, layerCount int) {
	if runtimeComp == nil || len(runtimeComp.LoadedLayers) >= layerCount {
		return
	}
	loadedLayers := make([]bool, layerCount)
	copy(loadedLayers, runtimeComp.LoadedLayers)
	runtimeComp.LoadedLayers = loadedLayers
}

func ensureLayerMeta(lvl *levels.Level, layerIndex int) {
	if lvl == nil || layerIndex < 0 {
		return
	}
	if len(lvl.LayerMeta) > layerIndex {
		return
	}
	meta := make([]levels.LayerMeta, layerIndex+1)
	copy(meta, lvl.LayerMeta)
	lvl.LayerMeta = meta
}

func setRuntimeLayerEntityState(world *ecs.World, layerIndex int, active bool) {
	disabled := !active
	ecs.ForEach(world, component.EntityLayerComponent.Kind(), func(e ecs.Entity, layer *component.EntityLayer) {
		if layer == nil || layer.Index != layerIndex {
			return
		}
		if sprite, ok := ecs.Get(world, e, component.SpriteComponent.Kind()); ok && sprite != nil {
			sprite.Disabled = disabled
		}
		if body, ok := ecs.Get(world, e, component.PhysicsBodyComponent.Kind()); ok && body != nil {
			body.Disabled = disabled
		}
		if hazard, ok := ecs.Get(world, e, component.HazardComponent.Kind()); ok && hazard != nil {
			hazard.Disabled = disabled
		}
		if circle, ok := ecs.Get(world, e, component.CircleRenderComponent.Kind()); ok && circle != nil {
			circle.Disabled = disabled
		}
		if input, ok := ecs.Get(world, e, component.InputComponent.Kind()); ok && input != nil {
			input.Disabled = disabled
		}
	})
	// Mark the static tile batch dirty when entity layer visibility changes.
	if b, ok := ecs.First(world, component.LevelGridComponent.Kind()); ok {
		if st, ok := ecs.Get(world, b, component.StaticTileBatchStateComponent.Kind()); ok && st != nil {
			st.Dirty = true
		}
	}
}

func rebuildLevelGrid(world *ecs.World, runtimeComp *component.LevelRuntime) error {
	grid, err := levelGrid(world)
	if err != nil {
		return err
	}
	tileSize := runtimeComp.TileSize
	if tileSize <= 0 {
		tileSize = grid.TileSize
	}
	if tileSize <= 0 {
		tileSize = 32
	}
	rebuilt := levelentity.BuildLevelGridData(runtimeComp.Level, tileSize)
	*grid = *rebuilt
	return nil
}

func findLevelLayerIndex(lvl *levels.Level, layerName string) int {
	if lvl == nil {
		return -1
	}
	needle := strings.TrimSpace(layerName)
	if needle == "" {
		return -1
	}
	for idx, meta := range lvl.LayerMeta {
		if strings.TrimSpace(meta.Name) == needle {
			return idx
		}
	}
	return -1
}

func layerNameArg(obj tengo.Object) (string, error) {
	value, ok := obj.(*tengo.String)
	if !ok || strings.TrimSpace(value.Value) == "" {
		return "", fmt.Errorf("layer name must be a non-empty string")
	}
	return value.Value, nil
}

func levelentityLayerHasPhysics(lvl *levels.Level, layerIndex int) bool {
	if lvl == nil || layerIndex < 0 || layerIndex >= len(lvl.LayerMeta) {
		return false
	}
	return lvl.LayerMeta[layerIndex].Physics
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
	if transform.Parent != 0 {
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
