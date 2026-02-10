package component

type AI struct {
	MoveSpeed    float64
	FollowRange  float64
	AttackRange  float64
	AttackFrames int
}

var AIComponent = NewComponent[AI]()
