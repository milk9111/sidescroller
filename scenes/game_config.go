package scenes

import (
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/internal/savegame"
)

type GameConfig struct {
	LevelName        string
	Debug            bool
	AllAbilities     bool
	WatchPrefabs     bool
	Overlay          bool
	Mute             bool
	InitialFadeIn    bool
	InitialAbilities *component.Abilities
	SaveStore        *savegame.Store
	LoadedSave       *savegame.File
}
