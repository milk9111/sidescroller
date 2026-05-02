package component

type Shrine struct {
	Range float64
}

var ShrineComponent = NewComponent[Shrine]()