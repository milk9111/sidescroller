package component

type Persistent struct {
	ID                string
	KeepOnLevelChange bool
	KeepOnReload      bool
}

var PersistentComponent = NewComponent[Persistent]()
