package component

type SpawnChildSpec struct {
	Prefab string
}

type SpawnChildren struct {
	Children []SpawnChildSpec
}

var SpawnChildrenComponent = NewComponent[SpawnChildren]()

type SpawnChildrenRuntime struct {
	Spawned map[string]uint64
}

var SpawnChildrenRuntimeComponent = NewComponent[SpawnChildrenRuntime]()
