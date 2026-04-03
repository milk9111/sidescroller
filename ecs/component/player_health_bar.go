package component

type PlayerHealthBar struct {
	MaxHearts     int
	LastHealth    int
	LastGearCount int
	LastHealUses  int
	LastCanHeal   bool
}

var PlayerHealthBarComponent = NewComponent[PlayerHealthBar]()
