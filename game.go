package main

import (
	"fmt"
	"log"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/obj"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	frames int

	input     *obj.Input
	player    *obj.Player
	level     *obj.Level
	camera    *obj.Camera
	debugDraw bool
}

func NewGame(levelPath string, debug bool) *Game {
	var lvl *obj.Level
	if levelPath != "" {
		if l, err := obj.LoadLevelFromFS(levels.LevelsFS, levelPath); err == nil {
			lvl = l
		} else if l, err := obj.LoadLevel(levelPath); err == nil {
			lvl = l
		} else {
			log.Printf("failed to load level %s: %v", levelPath, err)
		}
	}

	spawnX, spawnY := lvl.GetSpawnPosition()

	collisionWorld := obj.NewCollisionWorld(lvl)
	input := obj.NewInput()
	player := obj.NewPlayer(spawnX, spawnY, input, collisionWorld)
	g := &Game{
		input:     input,
		player:    player,
		level:     lvl,
		debugDraw: debug,
	}
	// create camera centered on player; default zoom 1.5
	g.camera = obj.NewCamera(common.BaseWidth, common.BaseHeight, 2)
	g.camera.SetWorldBounds(lvl.Width*common.TileSize, lvl.Height*common.TileSize)
	// initialize camera position to player's center to avoid large initial lerp
	cx := float64(player.X + float32(player.Width)/2.0)
	cy := float64(player.Y + float32(player.Height)/2.0)
	g.camera.PosX = cx
	g.camera.PosY = cy
	return g
}

func (g *Game) Update() error {
	g.frames++
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.debugDraw = !g.debugDraw
	}

	g.input.Update()
	// compute mouse world coordinates and mouse button presses
	mx, my := ebiten.CursorPosition()
	vx, vy := g.camera.ViewTopLeft()
	// world coordinates = view top-left + screen coords / zoom
	g.input.MouseWorldX = vx + float64(mx)/g.camera.Zoom()
	g.input.MouseWorldY = vy + float64(my)/g.camera.Zoom()
	g.input.MouseLeftPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	g.player.Update()
	cx := float64(g.player.X + float32(g.player.Width)/2.0)
	cy := float64(g.player.Y + float32(g.player.Height)/2.0)
	g.camera.Update(cx, cy)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Frames: %d    FPS: %.2f    State: %s    GravityEnabled: %g", g.frames, ebiten.ActualFPS(), g.player.GetState(), g.player.GravityEnabled))
	g.camera.Render(screen, func(world *ebiten.Image) {
		vx, vy := g.camera.ViewTopLeft()
		g.level.Draw(world, vx, vy)
		g.player.Draw(world)
		if g.debugDraw && g.player != nil && g.player.CollisionWorld != nil {
			g.player.CollisionWorld.DebugDraw(world)
		}
	})
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return common.BaseWidth, common.BaseHeight
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("shouldn't use Layout")
}
