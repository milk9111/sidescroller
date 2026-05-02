package component

// CheckpointReloadRequest asks the persistence system to reload from the last
// saved checkpoint snapshot. When SaveBeforeReload is true, the current world
// snapshot is saved first and then used for the reload.
type CheckpointReloadRequest struct {
	SaveBeforeReload bool
}

var CheckpointReloadRequestComponent = NewComponent[CheckpointReloadRequest]()