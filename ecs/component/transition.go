package component

// TransitionDirection indicates the direction the player is entering the next
// level from. It is authored in the editor as one of: up, down, left, right.
//
// NOTE: This is intentionally a string type so it serializes cleanly in JSON
// props and is easy to extend.
type TransitionDirection string

const (
	TransitionDirUp    TransitionDirection = "up"
	TransitionDirDown  TransitionDirection = "down"
	TransitionDirLeft  TransitionDirection = "left"
	TransitionDirRight TransitionDirection = "right"
)

// AABB is an axis-aligned bounding box.
// X/Y are offsets relative to the owning entity's Transform.
type AABB struct {
	X float64
	Y float64
	W float64
	H float64
}

// Transition defines a rectangular area that triggers a level change when the
// player enters it.
type Transition struct {
	// ID is this transition area's identifier within its level.
	ID string
	// TargetLevel is the level name to load when activated.
	TargetLevel string
	// LinkedID is the ID of the transition area in the target level that the
	// player should spawn into.
	LinkedID string
	// EnterDir is the direction the player is entering the new level from.
	EnterDir TransitionDirection
	// Bounds is the rectangle (in world units/pixels) used for overlap checks.
	Bounds AABB
}

var TransitionComponent = NewComponent[Transition]()
