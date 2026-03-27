package component

type PlayerHealthBar struct {
	MaxHearts     int
	LastHealth    int
	LastGearCount int
}

var PlayerHealthBarComponent = NewComponent[PlayerHealthBar]()
