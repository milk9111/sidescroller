package component

// RespawnRequest is a marker component indicating a player should be
// teleported to their safe respawn position. A system running after
// physics will perform the actual movement so physics constraints can be
// removed first.
type RespawnRequest struct{}

var RespawnRequestComponent = NewComponent[RespawnRequest]()
