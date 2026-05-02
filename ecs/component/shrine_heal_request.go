package component

// ShrineHealRequest asks the player controller to enter the shrine heal state.
type ShrineHealRequest struct{}

var ShrineHealRequestComponent = NewComponent[ShrineHealRequest]()
