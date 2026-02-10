package component

type PlayerTag struct{}

var PlayerTagComponent = NewComponent[PlayerTag]()

type CameraTag struct{}

var CameraTagComponent = NewComponent[CameraTag]()

type AimTargetTag struct{}

var AimTargetTagComponent = NewComponent[AimTargetTag]()

type AnchorTag struct{}

var AnchorTagComponent = NewComponent[AnchorTag]()
