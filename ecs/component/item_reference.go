package component

type ItemReference struct {
	Prefab string
}

var ItemReferenceComponent = NewComponent[ItemReference]()
