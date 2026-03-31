package scenes

import "github.com/milk9111/sidescroller/ecs/component"

type GameConfig struct {
	LevelName        string
	Debug            bool
	AllAbilities     bool
	WatchPrefabs     bool
	Overlay          bool
	Mute             bool
	InitialAbilities *component.Abilities
}
