package main

import (
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	baseWidth  = 1280
	baseHeight = 720
)

type Game struct {
	frames int

	input  *Input
	player *Player
	level  *Level
}

func NewGame(levelPath string) *Game {
	var lvl *Level
	if levelPath != "" {
		l, err := LoadLevel(levelPath)
		if err != nil {
			log.Printf("failed to load level %s: %v", levelPath, err)
		} else {
			lvl = l
		}
	}

	spawnX, spawnY := lvl.GetSpawnPosition()

	collisionWorld := NewCollisionWorld(lvl)
	input := NewInput()
	player := NewPlayer(spawnX, spawnY, input, collisionWorld)
	return &Game{
		input:  input,
		player: player,
		level:  lvl,
	}
}

func (g *Game) Update() error {
	g.frames++

	g.input.Update()
	g.player.Update()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Frames: %d    FPS: %.2f", g.frames, ebiten.ActualFPS()))

	if g.level != nil {
		g.level.Draw(screen)
	}

	g.player.Draw(screen)
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return baseWidth, baseHeight
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("shouldn't use Layout")
}
