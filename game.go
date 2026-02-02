package main

import (
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/obj"

	"github.com/ebitenui/ebitenui"
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
	anchor         *obj.Anchor

	debugDraw bool
	baseZoom  float64
	// recentlyTeleported prevents immediate retriggering of transitions
	recentlyTeleported bool
	// Transition manager (handles fade/load/fade)
	transition *obj.Transition
	ui         *ebitenui.UI
	paused     bool
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

	anchor := obj.NewAnchor()
	collisionWorld := obj.NewCollisionWorld(lvl)
	input := obj.NewInput(camera)
	player := obj.NewPlayer(spawnX, spawnY, input, collisionWorld, anchor, true)
	// let the input know about the player so it can provide gamepad aiming
	// (right stick) while in aim mode
	input.Player = player

	anchor.Init(player, camera, collisionWorld)

	// initialize camera position to player's center to avoid large initial lerp
	cx := float64(player.X + float32(player.Width)/2.0)
	cy := float64(player.Y + float32(player.Height)/2.0)
	camera.SnapTo(cx, cy)

	g := &Game{
		input:          input,
		player:         player,
		level:          lvl,
		debugDraw:      debug,
		camera:         camera,
		baseZoom:       baseZoom,
		collisionWorld: collisionWorld,
		anchor:         anchor,
		transition:     obj.NewTransition(),
	}

	// create pause UI
	g.ui = NewPauseUI(g)

	// wire transition callback to perform the actual level load and setup
	if g.transition != nil {
		g.transition.OnStart = func(target, linkID, direction string) {
			var newLvl *obj.Level
			if l, err := obj.LoadLevelFromFS(levels.LevelsFS, target); err == nil {
				newLvl = l
			} else if l, err := obj.LoadLevel(target); err == nil {
				newLvl = l
			} else {
				log.Printf("failed to load transition target %s: %v", target, err)
				newLvl = nil
			}

			if newLvl == nil {
				return
			}

			// determine spawn position
			var spawnX, spawnY float32
			spawnX, spawnY = newLvl.GetSpawnPosition()
			var targetTr *obj.TransitionData
			if linkID != "" {
				for i := range newLvl.Transitions {
					t2 := &newLvl.Transitions[i]
					if t2.ID == linkID {
						spawnX = float32(t2.X * common.TileSize)
						spawnY = float32(t2.Y * common.TileSize)
						targetTr = t2
						break
					}
				}
			}

			dir := "left"
			d := strings.ToLower(direction)
			switch d {
			case "up", "down", "left", "right":
				dir = d
			default:
				dir = "left"
			}

			if targetTr != nil {
				if dir == "up" || dir == "down" {
					centerX := float32(targetTr.X*common.TileSize) + float32(targetTr.W*common.TileSize)/2.0
					centerY := float32(targetTr.Y*common.TileSize) + float32(targetTr.H*common.TileSize)/2.0
					spawnX = centerX - 8.0
					spawnY = centerY - 20.0
				} else if dir == "left" || dir == "right" {
					// place player at bottom-center of the transition area
					centerX := float32(targetTr.X*common.TileSize) + float32(targetTr.W*common.TileSize)/2.0
					bottom := float32((targetTr.Y + targetTr.H) * common.TileSize)
					spawnX = centerX - 8.0
					spawnY = bottom - 40.0
				}
			}

			g.anchor = obj.NewAnchor()
			g.level = newLvl
			g.collisionWorld = obj.NewCollisionWorld(g.level)
			g.player = obj.NewPlayer(spawnX, spawnY, g.input, g.collisionWorld, g.anchor, g.player.IsFacingRight())
			g.anchor.Init(g.player, g.camera, g.collisionWorld)

			if strings.ToLower(direction) == "up" {
				g.player.ApplyTransitionJumpImpulse()
			}

			levelW := g.level.Width * common.TileSize
			levelH := g.level.Height * common.TileSize
			g.camera.SetWorldBounds(levelW, levelH)
			cx := float64(g.player.X + float32(g.player.Width)/2.0)
			cy := float64(g.player.Y + float32(g.player.Height)/2.0)
			g.camera.SnapTo(cx, cy)

			g.recentlyTeleported = true
		}
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

	// toggle pause on Esc
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.paused = !g.paused
	}

	// advance transition if active; transition.Update returns true while active
	if g.transition != nil && g.transition.Update() {
		return nil
	}

	// when paused, update UI and skip world updates
	if g.paused {
		if g.ui != nil {
			g.ui.Update()
		}
		return nil
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
		var hitTr *obj.TransitionData
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
				// start the transition via Transition manager
				if g.transition != nil {
					g.transition.Enter(hitTr.Target, hitTr.LinkID, hitTr.Direction)
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
		// determine whether the aiming ray hits a physics tile and
		// get the world coords of the ray end (hit point or target)
		px, py, hit := g.player.AimCollisionPoint(g.input.MouseWorldX, g.input.MouseWorldY)
		var img *ebiten.Image
		if hit && assets.AimTargetValid != nil {
			img = assets.AimTargetValid
		} else {
			img = assets.AimTargetInvalid
		}
		if img != nil {
			// convert world coords to screen space using camera
			vx, vy := g.camera.ViewTopLeft()
			zoom := g.camera.Zoom()
			sx := (px - vx) * zoom
			sy := (py - vy) * zoom
			w, h := img.Size()
			op := &ebiten.DrawImageOptions{}
			scale := 0.33
			op.GeoM.Scale(scale, scale)
			tx := sx - (float64(w)*scale)/2.0
			ty := sy - (float64(h)*scale)/2.0
			op.GeoM.Translate(tx, ty)
			op.Filter = ebiten.FilterLinear
			screen.DrawImage(img, op)
		}
	}

	// draw transition overlay (if active)
	if g.transition != nil {
		g.transition.Draw(screen)
	}

	// draw pause UI on top
	if g.paused && g.ui != nil {
		g.ui.Draw(screen)
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
