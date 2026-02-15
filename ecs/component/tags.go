package component

type PlayerTag struct{}

var PlayerTagComponent = NewComponent[PlayerTag]()

type CameraTag struct{}

var CameraTagComponent = NewComponent[CameraTag]()

type AimTargetTag struct{}

var AimTargetTagComponent = NewComponent[AimTargetTag]()

type AnchorTag struct{}

var AnchorTagComponent = NewComponent[AnchorTag]()

type SpikeTag struct{}

var SpikeTagComponent = NewComponent[SpikeTag]()

type AITag struct{}

var AITagComponent = NewComponent[AITag]()
