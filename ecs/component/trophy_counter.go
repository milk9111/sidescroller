package component

type TrophyCounter struct {
	RenderedText string
}

var TrophyCounterComponent = NewComponent[TrophyCounter]()

type TrophyCounterIcon struct{}

var TrophyCounterIconComponent = NewComponent[TrophyCounterIcon]()

type TrophyCounterText struct{}

var TrophyCounterTextComponent = NewComponent[TrophyCounterText]()
