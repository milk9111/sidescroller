package module

import (
	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
)

type Module struct {
	Name  string
	Build func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner ecs.Entity, target ecs.Entity) map[string]tengo.Object
}

func Builtins() []Module {
	return []Module{
		PrefabModule(),
		EntityModule(),
		ItemModule(),
		ScriptModule(),
		TransformModule(),
		LevelModule(),
		AudioModule(),
		AIModule(),
		AnimationModule(),
		PhysicsModule(),
		MusicModule(),
		InputModule(),
		DebugModule(),
		CameraModule(),
		SpriteModule(),
		ParticleEmitterModule(),
		PlayerModule(),
		HealthModule(),
		BreakableWallModule(),
		HazardModule(),
		ArenaModule(),
	}
}
