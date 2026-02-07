package component

type PlayerTag struct{}

var PlayerTagComponent = NewComponent[PlayerTag]()

type CameraTag struct{}

var CameraTagComponent = NewComponent[CameraTag]()
