package system

import (
	"container/heap"
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	defaultPathGridSize     = 32.0
	defaultPathRepathFrames = 15
	defaultDebugNodeSize    = 3.0
)

type PathfindingSystem struct{}

func NewPathfindingSystem() *PathfindingSystem {
	return &PathfindingSystem{}
}

func (ps *PathfindingSystem) Update(w *ecs.World) {
	if ps == nil || w == nil {
		return
	}

	playerX, playerY, playerFound := playerPosition(w)
	if !playerFound {
		return
	}

	bounds, ok := levelBounds(w)
	if !ok {
		return
	}

	ecs.ForEach(w, component.PathfindingComponent.Kind(), func(e ecs.Entity, pf *component.Pathfinding) {
		gridSize := defaultPathGridSize
		if pf.GridSize > 0 {
			gridSize = pf.GridSize
		} else {
			pf.GridSize = gridSize
		}

		gridW := int(math.Ceil(bounds.Width / gridSize))
		gridH := int(math.Ceil(bounds.Height / gridSize))
		if gridW <= 0 || gridH <= 0 {
			return
		}

		blocked := buildBlockedGrid(w, gridW, gridH, gridSize)

		if pf.RepathFrames <= 0 {
			pf.RepathFrames = defaultPathRepathFrames
		}
		if pf.DebugNodeSize <= 0 {
			pf.DebugNodeSize = defaultDebugNodeSize
		}

		startX, startY, ok := entityPosition(w, e)
		if !ok {
			return
		}

		start := gridCoord(startX, startY, gridSize, gridW, gridH)
		goal := gridCoord(playerX, playerY, gridSize, gridW, gridH)

		pf.FrameCounter++
		// if pf.FrameCounter%pf.RepathFrames != 0 &&
		// 	pf.LastStartX == start.x && pf.LastStartY == start.y &&
		// 	pf.LastTargetX == goal.x && pf.LastTargetY == goal.y &&
		// 	len(pf.Path) > 0 {
		// 	if err := ecs.Add(w, ecs.Entities(w)[e], component.PathfindingComponent.Kind(), pf); err != nil {
		// 		panic("pathfinding: update component: " + err.Error())
		// 	}
		// 	return
		// }

		path, visited := astarPath(start, goal, blocked, gridW, gridH)

		pf.Path = gridPathToWorld(path, gridSize)
		pf.Visited = gridPathToWorld(visited, gridSize)
		pf.LastStartX = start.x
		pf.LastStartY = start.y
		pf.LastTargetX = goal.x
		pf.LastTargetY = goal.y

		// if err := ecs.Add(w, ent, component.PathfindingComponent.Kind(), pf); err != nil {
		// 	panic("pathfinding: update component: " + err.Error())
		// }
	})
}

type gridPos struct {
	x int
	y int
}

func playerPosition(w *ecs.World) (float64, float64, bool) {
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return 0, 0, false
	}
	if t, ok := ecs.Get(w, player, component.TransformComponent.Kind()); ok {
		return t.X, t.Y, true
	}
	if pb, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind()); ok && pb.Body != nil {
		pos := pb.Body.Position()
		return pos.X, pos.Y, true
	}
	return 0, 0, false
}

func entityPosition(w *ecs.World, ent ecs.Entity) (float64, float64, bool) {
	if pb, ok := ecs.Get(w, ent, component.PhysicsBodyComponent.Kind()); ok && pb.Body != nil {
		pos := pb.Body.Position()
		return pos.X, pos.Y, true
	}
	if t, ok := ecs.Get(w, ent, component.TransformComponent.Kind()); ok {
		return t.X, t.Y, true
	}
	return 0, 0, false
}

func levelBounds(w *ecs.World) (component.LevelBounds, bool) {
	boundsEntity, ok := ecs.First(w, component.LevelBoundsComponent.Kind())
	if !ok {
		return component.LevelBounds{}, false
	}
	bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent.Kind())
	return *bounds, ok
}

func gridCoord(x, y, gridSize float64, gridW, gridH int) gridPos {
	gx := int(math.Floor(x / gridSize))
	gy := int(math.Floor(y / gridSize))
	if gx < 0 {
		gx = 0
	}
	if gy < 0 {
		gy = 0
	}
	if gx >= gridW {
		gx = gridW - 1
	}
	if gy >= gridH {
		gy = gridH - 1
	}
	return gridPos{x: gx, y: gy}
}

func gridPathToWorld(path []gridPos, gridSize float64) []component.PathNode {
	if len(path) == 0 {
		return nil
	}
	out := make([]component.PathNode, 0, len(path))
	half := gridSize * 0.5
	for _, p := range path {
		out = append(out, component.PathNode{
			X: float64(p.x)*gridSize + half,
			Y: float64(p.y)*gridSize + half,
		})
	}
	return out
}

func buildBlockedGrid(w *ecs.World, gridW, gridH int, gridSize float64) []bool {
	blocked := make([]bool, gridW*gridH)
	ecs.ForEach2(w, component.PhysicsBodyComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, body *component.PhysicsBody, transform *component.Transform) {
		if !body.Static {
			return
		}

		minX, minY, maxX, maxY := bodyAABBForPath(w, e, transform, body)
		startX := int(math.Floor(minX / gridSize))
		startY := int(math.Floor(minY / gridSize))
		endX := int(math.Floor((maxX - 0.001) / gridSize))
		endY := int(math.Floor((maxY - 0.001) / gridSize))

		if startX < 0 {
			startX = 0
		}
		if startY < 0 {
			startY = 0
		}
		if endX >= gridW {
			endX = gridW - 1
		}
		if endY >= gridH {
			endY = gridH - 1
		}

		for y := startY; y <= endY; y++ {
			for x := startX; x <= endX; x++ {
				idx := y*gridW + x
				if idx >= 0 && idx < len(blocked) {
					blocked[idx] = true
				}
			}
		}
	})

	return blocked
}

func bodyAABBForPath(w *ecs.World, e ecs.Entity, transform *component.Transform, body *component.PhysicsBody) (minX, minY, maxX, maxY float64) {
	width := body.Width
	height := body.Height
	if width <= 0 {
		width = 32
	}
	if height <= 0 {
		height = 32
	}

	if body.AlignTopLeft {
		minX = aabbTopLeftX(w, e, transform.X, body.OffsetX, width, true)
		minY = transform.Y + body.OffsetY
	} else {
		minX = aabbTopLeftX(w, e, transform.X, body.OffsetX, width, false)
		minY = transform.Y + body.OffsetY - height/2
	}
	maxX = minX + width
	maxY = minY + height
	return
}

func astarPath(start, goal gridPos, blocked []bool, gridW, gridH int) ([]gridPos, []gridPos) {
	if start.x < 0 || start.y < 0 || goal.x < 0 || goal.y < 0 {
		return nil, nil
	}
	if start.x >= gridW || start.y >= gridH || goal.x >= gridW || goal.y >= gridH {
		return nil, nil
	}
	if blocked[start.y*gridW+start.x] || blocked[goal.y*gridW+goal.x] {
		return nil, nil
	}

	open := &openSet{}
	heap.Init(open)

	cameFrom := make([]int, gridW*gridH)
	for i := range cameFrom {
		cameFrom[i] = -1
	}
	gScore := make([]float64, gridW*gridH)
	for i := range gScore {
		gScore[i] = math.Inf(1)
	}
	startIdx := start.y*gridW + start.x
	goalIdx := goal.y*gridW + goal.x
	gScore[startIdx] = 0
	heap.Push(open, &openItem{pos: start, f: heuristic(start, goal), g: 0})

	visited := make([]gridPos, 0, 64)

	for open.Len() > 0 {
		current := heap.Pop(open).(*openItem)
		cur := current.pos
		curIdx := cur.y*gridW + cur.x

		visited = append(visited, cur)

		if curIdx == goalIdx {
			return reconstructPath(cameFrom, gridW, startIdx, goalIdx), visited
		}

		for _, n := range neighbors(cur, gridW, gridH) {
			idx := n.y*gridW + n.x
			if blocked[idx] {
				continue
			}
			tentativeG := gScore[curIdx] + 1
			if tentativeG < gScore[idx] {
				cameFrom[idx] = curIdx
				gScore[idx] = tentativeG
				f := tentativeG + heuristic(n, goal)
				heap.Push(open, &openItem{pos: n, f: f, g: tentativeG})
			}
		}
	}

	return nil, visited
}

func reconstructPath(cameFrom []int, gridW int, startIdx, goalIdx int) []gridPos {
	if startIdx == goalIdx {
		return []gridPos{{x: startIdx % gridW, y: startIdx / gridW}}
	}
	if goalIdx < 0 || goalIdx >= len(cameFrom) || cameFrom[goalIdx] == -1 {
		return nil
	}

	path := make([]gridPos, 0, 32)
	cur := goalIdx
	for cur != -1 {
		x := cur % gridW
		y := cur / gridW
		path = append(path, gridPos{x: x, y: y})
		if cur == startIdx {
			break
		}
		cur = cameFrom[cur]
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func neighbors(p gridPos, gridW, gridH int) []gridPos {
	out := make([]gridPos, 0, 4)
	if p.x > 0 {
		out = append(out, gridPos{x: p.x - 1, y: p.y})
	}
	if p.x < gridW-1 {
		out = append(out, gridPos{x: p.x + 1, y: p.y})
	}
	if p.y > 0 {
		out = append(out, gridPos{x: p.x, y: p.y - 1})
	}
	if p.y < gridH-1 {
		out = append(out, gridPos{x: p.x, y: p.y + 1})
	}
	return out
}

func heuristic(a, b gridPos) float64 {
	return math.Abs(float64(a.x-b.x)) + math.Abs(float64(a.y-b.y))
}

type openItem struct {
	pos   gridPos
	f     float64
	g     float64
	index int
}

type openSet []*openItem

func (o openSet) Len() int           { return len(o) }
func (o openSet) Less(i, j int) bool { return o[i].f < o[j].f }
func (o openSet) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
	o[i].index = i
	o[j].index = j
}
func (o *openSet) Push(x any) {
	item := x.(*openItem)
	item.index = len(*o)
	*o = append(*o, item)
}
func (o *openSet) Pop() any {
	old := *o
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*o = old[:n-1]
	return item
}
