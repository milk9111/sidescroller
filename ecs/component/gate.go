package component

// Gate marks an entity as a gate controlled by arena activation toggles.
type Gate struct{}

// GateRuntime caches authored components so gates can be disabled/enabled
// without destroying the entity.
type GateRuntime struct {
	Initialized     bool
	HasSprite       bool
	SpriteTemplate  Sprite
	HasPhysicsBody  bool
	PhysicsTemplate PhysicsBody
}

var GateComponent = NewComponent[Gate]()
var GateRuntimeComponent = NewComponent[GateRuntime]()
