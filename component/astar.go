package component

import "math"

// PathNode represents a grid cell in an A* path.
type PathNode struct {
	X int
	Y int
}

// AStar finds a path from start to goal on a 4-way grid.
// isBlocked should return true for cells that cannot be traversed.
// maxNodes limits the number of processed nodes to avoid runaway searches.
func AStar(startX, startY, goalX, goalY, width, height int, isBlocked func(x, y int) bool, maxNodes int) []PathNode {
	if width <= 0 || height <= 0 {
		return nil
	}
	if startX == goalX && startY == goalY {
		return []PathNode{{X: startX, Y: startY}}
	}
	if goalX < 0 || goalY < 0 || goalX >= width || goalY >= height {
		return nil
	}
	if isBlocked != nil && isBlocked(goalX, goalY) {
		return nil
	}

	startIdx := startY*width + startX
	goalIdx := goalY*width + goalX

	open := make([]PathNode, 0, 64)
	open = append(open, PathNode{X: startX, Y: startY})
	openSet := map[int]bool{startIdx: true}

	cameFrom := make(map[int]int, 128)
	gScore := make(map[int]float64, 128)
	fScore := make(map[int]float64, 128)
	gScore[startIdx] = 0
	fScore[startIdx] = heuristic(startX, startY, goalX, goalY)

	iterations := 0
	for len(open) > 0 && iterations < maxNodes {
		iterations++
		// find node with lowest fScore
		bestIdx := 0
		bestScore := math.MaxFloat64
		for i, n := range open {
			idx := n.Y*width + n.X
			if f, ok := fScore[idx]; ok && f < bestScore {
				bestScore = f
				bestIdx = i
			}
		}
		current := open[bestIdx]
		currentIdx := current.Y*width + current.X
		// remove from open
		open = append(open[:bestIdx], open[bestIdx+1:]...)
		delete(openSet, currentIdx)

		if currentIdx == goalIdx {
			return reconstructPath(cameFrom, currentIdx, startIdx, width)
		}

		neighbors := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, d := range neighbors {
			nx := current.X + d[0]
			ny := current.Y + d[1]
			if nx < 0 || ny < 0 || nx >= width || ny >= height {
				continue
			}
			if isBlocked != nil && isBlocked(nx, ny) {
				continue
			}
			neighborIdx := ny*width + nx
			tentative := gScore[currentIdx] + 1
			prev, seen := gScore[neighborIdx]
			if !seen || tentative < prev {
				cameFrom[neighborIdx] = currentIdx
				gScore[neighborIdx] = tentative
				fScore[neighborIdx] = tentative + heuristic(nx, ny, goalX, goalY)
				if !openSet[neighborIdx] {
					open = append(open, PathNode{X: nx, Y: ny})
					openSet[neighborIdx] = true
				}
			}
		}
	}

	return nil
}

func reconstructPath(cameFrom map[int]int, currentIdx, startIdx, width int) []PathNode {
	path := make([]PathNode, 0, 32)
	for {
		x := currentIdx % width
		y := currentIdx / width
		path = append(path, PathNode{X: x, Y: y})
		if currentIdx == startIdx {
			break
		}
		prev, ok := cameFrom[currentIdx]
		if !ok {
			return nil
		}
		currentIdx = prev
	}
	// reverse
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func heuristic(x1, y1, x2, y2 int) float64 {
	return math.Abs(float64(x1-x2)) + math.Abs(float64(y1-y2))
}
