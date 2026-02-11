package component

// PathNode represents a world-space point along a path.
type PathNode struct {
	X float64
	Y float64
}

// Pathfinding stores grid-based pathfinding results and settings.
type Pathfinding struct {
	GridSize      float64
	RepathFrames  int
	FrameCounter  int
	LastStartX    int
	LastStartY    int
	LastTargetX   int
	LastTargetY   int
	Path          []PathNode
	Visited       []PathNode
	DebugNodeSize float64
}

var PathfindingComponent = NewComponent[Pathfinding]()
