package component

type TrophyCounter struct {
	Collected    int
	Total        int
	RenderedText string
}

var TrophyCounterComponent = NewComponent[TrophyCounter]()

type TrophyCounterIcon struct{}

var TrophyCounterIconComponent = NewComponent[TrophyCounterIcon]()

type TrophyCounterText struct{}

var TrophyCounterTextComponent = NewComponent[TrophyCounterText]()
