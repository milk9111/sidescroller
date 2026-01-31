package main

import (
	"fmt"
	"log"
	"math"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/obj"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	frames int

	input          *obj.Input
	player         *obj.Player
	level          *obj.Level
	camera         *obj.Camera
	collisionWorld *obj.CollisionWorld
	debugDraw      bool
	baseZoom       float64
	// recentlyTeleported prevents immediate retriggering of transitions
	recentlyTeleported bool
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

	levelW := lvl.Width * common.TileSize
	levelH := lvl.Height * common.TileSize
	baseZoom := 2.0
	camera := obj.NewCamera(common.BaseWidth, common.BaseHeight, baseZoom)
	camera.SetWorldBounds(levelW, levelH)

	spawnX, spawnY := lvl.GetSpawnPosition()

	collisionWorld := obj.NewCollisionWorld(lvl)
	input := obj.NewInput(camera)
	player := obj.NewPlayer(spawnX, spawnY, input, collisionWorld)

	// initialize camera position to player's center to avoid large initial lerp
	cx := float64(player.X + float32(player.Width)/2.0)
	cy := float64(player.Y + float32(player.Height)/2.0)
	camera.PosX = cx
	camera.PosY = cy

	g := &Game{
		input:          input,
		player:         player,
		level:          lvl,
		debugDraw:      debug,
		camera:         camera,
		baseZoom:       baseZoom,
		collisionWorld: collisionWorld,
		// physics time-scaling handled by player when aiming
	}

	return g
}

func (g *Game) Update() error {
	g.frames++
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		g.debugDraw = !g.debugDraw
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.baseZoom += 0.1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.baseZoom -= 0.1
		if g.baseZoom < 0.1 {
			g.baseZoom = 0.1
		}
	}

	g.collisionWorld.Update()
	g.input.Update()

	// perform physics step early so collision/contact flags are available
	// to the player's Update/OnPhysics logic in the same frame. Use player's
	// PhysicsTimeScale (set by aiming state) if available.
	dt := 1.0
	if g.player != nil && g.player.PhysicsTimeScale > 0 {
		dt = g.player.PhysicsTimeScale
	}
	g.collisionWorld.Step(dt)

	g.player.Update()

	cx := float64(g.player.X + float32(g.player.Width)/2.0)
	cy := float64(g.player.Y + float32(g.player.Height)/2.0)
	g.camera.Update(cx, cy)

	// handle level transitions: if player overlaps a transition rect, load target level
	if g.level != nil && g.player != nil {
		// compute player's occupied tile bounds
		left := int(math.Floor(float64(g.player.X) / float64(common.TileSize)))
		top := int(math.Floor(float64(g.player.Y) / float64(common.TileSize)))
		right := int(math.Floor(float64(g.player.X+float32(g.player.Width)-1) / float64(common.TileSize)))
		bottom := int(math.Floor(float64(g.player.Y+float32(g.player.Height)-1) / float64(common.TileSize)))

		overlapping := false
		var hitTr *obj.Transition
		for i := range g.level.Transitions {
			tr := &g.level.Transitions[i]
			if right < tr.X || left > tr.X+tr.W-1 || bottom < tr.Y || top > tr.Y+tr.H-1 {
				continue
			}
			overlapping = true
			hitTr = tr
			break
		}

		if overlapping {
			if !g.recentlyTeleported && hitTr != nil && hitTr.Target != "" {
				// attempt to load the target level (prefer embedded FS then disk)
				var newLvl *obj.Level
				if l, err := obj.LoadLevelFromFS(levels.LevelsFS, hitTr.Target); err == nil {
					newLvl = l
				} else if l, err := obj.LoadLevel(hitTr.Target); err == nil {
					newLvl = l
				} else {
					// failed to load target; log and skip
					log.Printf("failed to load transition target %s: %v", hitTr.Target, err)
					newLvl = nil
				}

				if newLvl != nil {
					// find target transition in new level matching LinkID
					var spawnX, spawnY float32
					spawnX, spawnY = newLvl.GetSpawnPosition()
					if hitTr.LinkID != "" {
						for i := range newLvl.Transitions {
							t2 := &newLvl.Transitions[i]
							// match target transition by its ID (the source transition's LinkID points to the target's ID)
							if t2.ID == hitTr.LinkID {
								// position player at the top-left of the linked transition rect
								spawnX = float32(t2.X * common.TileSize)
								spawnY = float32(t2.Y * common.TileSize)
								break
							}
						}
					}

					// switch level and recreate collision world + player
					g.level = newLvl
					g.collisionWorld = obj.NewCollisionWorld(g.level)
					// create a new player at spawnX/spawnY using existing input
					g.player = obj.NewPlayer(spawnX, spawnY, g.input, g.collisionWorld)

					// update camera bounds to new level size and center on player
					levelW := g.level.Width * common.TileSize
					levelH := g.level.Height * common.TileSize
					g.camera.SetWorldBounds(levelW, levelH)
					cx = float64(g.player.X + float32(g.player.Width)/2.0)
					cy = float64(g.player.Y + float32(g.player.Height)/2.0)
					g.camera.PosX = cx
					g.camera.PosY = cy

					// mark recently teleported to avoid immediate re-trigger
					g.recentlyTeleported = true
				}
			}
		} else {
			// not overlapping any transition, clear teleport flag so re-entry will trigger
			g.recentlyTeleported = false
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Frames: %d    FPS: %.2f    State: %s    GravityEnabled: %g", g.frames, ebiten.ActualFPS(), g.player.GetState(), g.player.GravityEnabled))
	g.camera.Render(screen, func(world *ebiten.Image) {
		vx, vy := g.camera.ViewTopLeft()
		zoom := g.camera.Zoom()
		g.level.Draw(world, vx, vy, zoom)
		g.player.Draw(world, vx, vy, zoom)
		if g.debugDraw && g.player != nil && g.player.CollisionWorld != nil {
			g.player.CollisionWorld.DebugDraw(world, vx, vy, zoom)
		}
	})

	// draw cursor replacement in screen space while aiming
	if g.player != nil && g.player.IsAiming() && g.input != nil {
		// determine whether the aiming ray hits a physics tile
		_, _, hit := g.player.AimCollisionPoint(g.input.MouseWorldX, g.input.MouseWorldY)
		var img *ebiten.Image
		if hit && assets.AimTargetValid != nil {
			img = assets.AimTargetValid
		} else {
			img = assets.AimTargetInvalid
		}
		if img != nil {
			mx, my := ebiten.CursorPosition()
			w, h := img.Size()
			op := &ebiten.DrawImageOptions{}
			scale := 0.33
			op.GeoM.Scale(scale, scale)
			tx := float64(mx) - (float64(w)*scale)/2.0
			ty := float64(my) - (float64(h)*scale)/2.0
			op.GeoM.Translate(tx, ty)
			op.Filter = ebiten.FilterLinear
			screen.DrawImage(img, op)
		}
	}
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	if g.camera != nil {
		g.camera.SetScreenSize(int(outsideWidth), int(outsideHeight))
		if g.level != nil {
			worldW := float64(g.level.Width * common.TileSize)
			worldH := float64(g.level.Height * common.TileSize)
			if worldW > 0 && worldH > 0 {
				minZoom := math.Max(outsideWidth/worldW, outsideHeight/worldH)
				zoom := g.baseZoom
				if zoom < minZoom {
					zoom = minZoom
				}
				g.camera.SetZoom(zoom)
			}
		}
	}
	return outsideWidth, outsideHeight
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("shouldn't use Layout")
}
