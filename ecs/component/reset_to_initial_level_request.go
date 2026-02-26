package component

// ResetToInitialLevelRequest asks the persistence system to reset to the
// configured initial level and perform a full reload.
type ResetToInitialLevelRequest struct{}

var ResetToInitialLevelRequestComponent = NewComponent[ResetToInitialLevelRequest]()
