package component

type PlayerHealthBar struct {
	MaxHearts int
}

var PlayerHealthBarComponent = NewComponent[PlayerHealthBar]()

type PlayerHealthHeart struct {
	Slot int
}

var PlayerHealthHeartComponent = NewComponent[PlayerHealthHeart]()
