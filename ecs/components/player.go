package components

import "github.com/milk9111/sidescroller/component"

// PlayerController stores player movement and attack state.
type PlayerController struct {
	MoveSpeed    float32
	JumpVelocity float32
	MaxSpeedX    float32
	FacingRight  bool
	State        string
	AttackFrames int
	AttackTimer  int
	CoyoteFrames int
	CoyoteTimer  int
	JumpsUsed    int
	MaxJumps     int
	DoubleJump   bool
	WallGrab     bool
	Swing        bool
	Dash         bool
	IdleAnim     *component.Animation
	RunAnim      *component.Animation
	AttackAnim   *component.Animation
}
