package component

// HitFreezeRequest requests a short global gameplay freeze measured in frames.
// Systems emit this data-only component and the outer game loop applies it.
type HitFreezeRequest struct {
	Frames int
}

var HitFreezeRequestComponent = NewComponent[HitFreezeRequest]()
