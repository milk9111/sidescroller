package component

type GameEntityID struct {
	Value string
}

var GameEntityIDComponent = NewComponent[GameEntityID]()
