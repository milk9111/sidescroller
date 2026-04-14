package component

type LeverHitRequest struct {
	SourceEntity uint64
}

var LeverHitRequestComponent = NewComponent[LeverHitRequest]()
