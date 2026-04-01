package component

type InventoryItem struct {
	Prefab string
	Count  int
}

type Inventory struct {
	Items []InventoryItem
}

var InventoryComponent = NewComponent[Inventory]()
