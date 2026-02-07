package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/system"
)

type Game struct {
	frames int
	world  *ecs.World
	render *system.RenderSystem
}

func NewGame(levelPath string, debug bool, allAbilities bool) *Game {
	world := ecs.NewWorld()
	render := system.NewRenderSystem(component.TransformComponent, component.SpriteComponent)

	player := world.CreateEntity()
	_ = ecs.Add(world, player, component.TransformComponent, component.Transform{X: 100, Y: 100})
	_ = ecs.Add(world, player, component.SpriteComponent, component.Sprite{Image: assets.PlayerV2Sheet})

	return &Game{
		world:  world,
		render: render,
	}
}

func (g *Game) Update() error {
	g.frames++

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
