package component

// ReloadRequest is a marker component used to signal the game loop to reload
// the current level/world. Systems may create a short-lived entity with this
// component to request a reload.
type ReloadRequest struct{}

var ReloadRequestComponent = NewComponent[ReloadRequest]()
