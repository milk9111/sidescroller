package component

type Trigger struct {
	Bounds   AABB
	Name     string
	Disabled bool
}

var TriggerComponent = NewComponent[Trigger]()
