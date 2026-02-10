package component

// AnchorConstraintMode indicates which constraint the physics system should apply.
const (
	AnchorConstraintSlide = "slide"
	AnchorConstraintPivot = "pivot"
	AnchorConstraintPin   = "pin"
)

// AnchorConstraintRequest asks the physics system to create or update
// anchor constraints for the player.
type AnchorConstraintRequest struct {
	Mode    string
	AnchorX float64
	AnchorY float64
	MinLen  float64
	MaxLen  float64
	Applied bool
}

var AnchorConstraintRequestComponent = NewComponent[AnchorConstraintRequest]()
