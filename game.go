package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/ecs/system"
	"github.com/milk9111/sidescroller/prefabs"
)

type Game struct {
	frames        int
	world         *ecs.World
	scheduler     *ecs.Scheduler
	render        *system.RenderSystem
	prefabWatcher *prefabs.Watcher
}

func NewGame(levelPath string, debug bool, allAbilities bool) *Game {
	game := &Game{
		world:     ecs.NewWorld(),
		scheduler: ecs.NewScheduler(),
		render:    system.NewRenderSystem(component.TransformComponent, component.SpriteComponent),
	}

	game.scheduler.Add(system.NewAnimationSystem(component.AnimationComponent, component.SpriteComponent))

	_ = game.reloadWorld()
	if watcher, err := prefabs.NewWatcher("prefabs"); err == nil {
		game.prefabWatcher = watcher
	}
	return game
}

func (g *Game) Update() error {
	g.frames++

	g.scheduler.Update(g.world)

	_ = g.processPrefabEvents()

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
	if _, err := entity.NewPlayer(world); err != nil {
		return err
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
