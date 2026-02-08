package main

import (
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/ecs/system"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/prefabs"
)

type Game struct {
	frames        int
	world         *ecs.World
	scheduler     *ecs.Scheduler
	render        *system.RenderSystem
	prefabWatcher *prefabs.Watcher
	levelName     string
}

func NewGame(levelName string, debug bool, allAbilities bool) *Game {
	game := &Game{
		world:     ecs.NewWorld(),
		scheduler: ecs.NewScheduler(),
		render:    system.NewRenderSystem(),
		levelName: levelName,
	}

	// Add systems in the order they should update
	game.scheduler.Add(system.NewAnimationSystem())
	game.scheduler.Add(system.NewPhysicsSystem())
	game.scheduler.Add(system.NewCameraSystem())

	if err := game.reloadWorld(); err != nil {
		panic("failed to load world: " + err.Error())
	}

	watcher, err := prefabs.NewWatcher("prefabs")
	if err != nil {
		panic("failed to create prefab watcher: " + err.Error())
	}

	game.prefabWatcher = watcher

	return game
}

func (g *Game) Update() error {
	g.frames++

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		os.Exit(0)
	}

	g.scheduler.Update(g.world)

	if err := g.processPrefabEvents(); err != nil {
		panic("failed to process prefab events: " + err.Error())
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.render != nil {
		g.render.Draw(g.world, screen)
	}
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return outsideWidth, outsideHeight
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("shouldn't use Layout")
}

func (g *Game) reloadWorld() error {
	world := ecs.NewWorld()

	name := g.levelName
	if filepath.Ext(name) == "" {
		name += ".json"
	}

	level, err := levels.LoadLevelFromFS(name)
	if err != nil {
		return err
	}

	if err = entity.LoadLevelToWorld(world, level); err != nil {
		return err
	}

	if _, err = entity.NewCamera(world); err != nil {
		return err
	}

	if len(level.Entities) == 0 {
		if _, err = entity.NewPlayer(world); err != nil {
			return err
		}
	}

	g.world = world
	return nil
}

func (g *Game) processPrefabEvents() error {
	if g.prefabWatcher == nil {
		return nil
	}

	reload := false
	for {
		select {
		case <-g.prefabWatcher.Events:
			reload = true
		case <-g.prefabWatcher.Errors:
			// Ignore errors for now; keep running.
		default:
			if reload {
				return g.reloadWorld()
			}
			return nil
		}
	}
}
