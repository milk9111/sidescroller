package component

type PlayerGearCount struct {
	Count int
}

var PlayerGearCountComponent = NewComponent[PlayerGearCount]()
