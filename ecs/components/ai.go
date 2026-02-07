package components

import "github.com/milk9111/sidescroller/component"

// AIKind identifies the AI behavior type.
type AIKind int

const (
	AIKindGroundEnemy AIKind = iota
	AIKindFlyingEnemy
)

// AIState stores behavior tuning and target info.
type AIState struct {
	Kind                AIKind
	TargetEntity        int
	AggroRange          float64
	AttackRange         float64
	AttackCooldown      int
	AttackCooldownTimer int
	MoveSpeed           float32
	AttackYOffset       float32
	AttackAlignDist     float32
	FacingRight         bool
}

// Pathfinding stores A* path state.
type Pathfinding struct {
	Path              []component.PathNode
	PathIndex         int
	RecalcTimer       int
	RecalcFrames      int
	MaxNodes          int
	WaypointReachDist float32
	LastGoalX         int
	LastGoalY         int
}
