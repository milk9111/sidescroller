package main

import (
	"fmt"
	"image/color"
	"log"
	"math"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/obj"
	"github.com/milk9111/sidescroller/system"

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
	pickups        []*obj.Pickup
	enemies        []*obj.Enemy
	flyingEnemies  []*obj.FlyingEnemy
	camera         *obj.Camera
	collisionWorld *obj.CollisionWorld
	anchor         *obj.Anchor
	world          *system.World

	debugDraw bool
	baseZoom  float64
	// recentlyTeleported prevents immediate retriggering of transitions
	recentlyTeleported bool
	// Transition manager (handles fade/load/fade)
	transition *obj.Transition
	ui         *ebitenui.UI

	paused bool
}

func NewGame(levelPath string, debug bool, allAbilities bool) *Game {
	var world *system.World
	if levelPath != "" {
		if w, err := system.NewWorld(levelPath); err == nil {
			world = w
		} else {
			log.Printf("failed to load level %s: %v", levelPath, err)
		}
	}
	var lvl *obj.Level
	if world != nil {
		lvl = world.Level
	}

	levelW := lvl.Width * common.TileSize
	levelH := lvl.Height * common.TileSize
	baseZoom := 2.0
	camera := obj.NewCamera(common.BaseWidth, common.BaseHeight, baseZoom)
	camera.SetWorldBounds(levelW, levelH)

	spawnX, spawnY := lvl.GetSpawnPosition()

	anchor := obj.NewAnchor()
	collisionWorld := world.CollisionWorld
	input := obj.NewInput(camera)

	doubleJumpEnabled := false
	wallGrabEnabled := false
	swingEnabled := false
	dashEnabled := false
	if allAbilities {
		doubleJumpEnabled = true
		wallGrabEnabled = true
		swingEnabled = true
		dashEnabled = true
	}

	player := obj.NewPlayer(spawnX, spawnY, input, collisionWorld, anchor, true, doubleJumpEnabled, wallGrabEnabled, swingEnabled, dashEnabled)
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
		world:          world,
	}

	// create pause UI
	g.ui = NewPauseUI(g)

	// shake camera when player takes damage
	if g.player != nil && g.player.Health() != nil {
		hp := g.player.Health()
		log.Printf("Assigning OnDamage for player health ptr=%p", hp)
		hp.OnDamage = func(h *component.Health, evt component.CombatEvent) {
			// log damage for debugging and start camera shake
			log.Printf("Player took damage: amt=%.2f attacker=%d target=%d frame=%d", evt.Damage, evt.AttackerID, evt.TargetID, evt.Frame)
			if g.camera != nil {
				// magnitude in pixels, duration in frames
				g.camera.StartShake(6.0, 12)
			}
		}
	}

	// spawn pickups/enemies from placed entities and remove them from level.Entities
	if g.world != nil && g.player != nil {
		g.world.SpawnEntities(g.player)
		g.pickups = g.world.Pickups
		g.enemies = g.world.Enemies
		g.flyingEnemies = g.world.FlyingEnemies
	}

	// wire transition callback to perform the actual level load and setup
	if g.transition != nil {
		g.transition.OnStart = func(target, linkID, direction string) {
			if g.world == nil {
				return
			}
			newPlayer, newAnchor, err := g.world.HandleTransition(target, linkID, direction, g.input, g.camera, g.player)
			if err != nil {
				log.Printf("failed to load transition target %s: %v", target, err)
				return
			}
			g.player = newPlayer
			g.anchor = newAnchor
			g.level = g.world.Level
			g.collisionWorld = g.world.CollisionWorld
			g.pickups = g.world.Pickups
			g.enemies = g.world.Enemies
			g.flyingEnemies = g.world.FlyingEnemies
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

	// update pickups (if present)
	if len(g.pickups) > 0 {
		for _, pickup := range g.pickups {
			if pickup != nil {
				pickup.Update(g.player)
			}
		}
	}

	// update enemies (if present)
	if len(g.enemies) > 0 {
		for _, enemy := range g.enemies {
			if enemy != nil {
				enemy.Update(g.player)
			}
		}
	}

	// update flying enemies (if present)
	if len(g.flyingEnemies) > 0 {
		for _, enemy := range g.flyingEnemies {
			if enemy != nil {
				enemy.Update(g.player)
			}
		}
	}

	system.ResolveCombat(g.player, g.enemies, g.camera)

	// tick global highlight store
	component.TickHighlights()

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
		if len(g.enemies) > 0 {
			for _, enemy := range g.enemies {
				if enemy != nil {
					enemy.Draw(world, vx, vy, zoom)
				}
			}
		}
		if len(g.flyingEnemies) > 0 {
			for _, enemy := range g.flyingEnemies {
				if enemy != nil {
					enemy.Draw(world, vx, vy, zoom)
				}
			}
		}
		g.player.Draw(world, vx, vy, zoom)

		if len(g.pickups) > 0 {
			for _, pickup := range g.pickups {
				if pickup != nil {
					pickup.Draw(world, vx, vy, zoom)
				}
			}
		}

		if g.debugDraw && g.player != nil && g.player.CollisionWorld != nil {
			g.player.CollisionWorld.DebugDraw(world, vx, vy, zoom)
			if len(g.enemies) > 0 {
				for _, enemy := range g.enemies {
					if enemy != nil {
						enemy.DrawDebugPath(world, vx, vy, zoom)
					}
				}
			}
			if len(g.flyingEnemies) > 0 {
				for _, enemy := range g.flyingEnemies {
					if enemy != nil {
						enemy.DrawDebugPath(world, vx, vy, zoom)
					}
				}
			}

			// gather all hitboxes and hurtboxes
			type boxRef struct {
				Owner  int
				ID     string
				Rect   common.Rect
				Active bool
				IsHit  bool
			}
			hitboxes := make([]boxRef, 0)
			hurtboxes := make([]boxRef, 0)
			if g.player != nil {
				for _, hb := range g.player.Hitboxes() {
					if !hb.Active {
						continue
					}
					hitboxes = append(hitboxes, boxRef{Owner: hb.OwnerID, ID: hb.ID, Rect: hb.Rect, Active: hb.Active, IsHit: true})
				}
				for _, hu := range g.player.Hurtboxes() {
					if !hu.Enabled {
						continue
					}
					hurtboxes = append(hurtboxes, boxRef{Owner: hu.OwnerID, ID: hu.ID, Rect: hu.Rect, Active: hu.Enabled, IsHit: false})
				}
			}
			for _, enemy := range g.enemies {
				if enemy == nil {
					continue
				}
				for _, hb := range enemy.Hitboxes() {
					if !hb.Active {
						continue
					}
					hitboxes = append(hitboxes, boxRef{Owner: hb.OwnerID, ID: hb.ID, Rect: hb.Rect, Active: hb.Active, IsHit: true})
				}
				for _, hu := range enemy.Hurtboxes() {
					if !hu.Enabled {
						continue
					}
					hurtboxes = append(hurtboxes, boxRef{Owner: hu.OwnerID, ID: hu.ID, Rect: hu.Rect, Active: hu.Enabled, IsHit: false})
				}
			}

			// detect collisions between hitboxes and hurtboxes
			collidedHit := make(map[int]bool)
			collidedHurt := make(map[int]bool)
			for i, hi := range hitboxes {
				for j, hu := range hurtboxes {
					if hi.Owner == hu.Owner {
						continue
					}
					if (&hi.Rect).Intersects(&hu.Rect) {
						collidedHit[i] = true
						collidedHurt[j] = true
					}
				}
			}

			// draw hurtboxes
			for j, hu := range hurtboxes {
				x := (float64(hu.Rect.X) - vx) * zoom
				y := (float64(hu.Rect.Y) - vy) * zoom
				w := float64(hu.Rect.Width) * zoom
				h := float64(hu.Rect.Height) * zoom
				col := color.RGBA{G: 0xff, A: 0xff}
				if collidedHurt[j] {
					col = color.RGBA{R: 0xff, A: 0xff}
				}
				ebitenutil.DrawLine(world, x, y, x+w, y, col)
				ebitenutil.DrawLine(world, x+w, y, x+w, y+h, col)
				ebitenutil.DrawLine(world, x+w, y+h, x, y+h, col)
				ebitenutil.DrawLine(world, x, y+h, x, y, col)
			}

			// draw hitboxes
			for _, hi := range hitboxes {
				x := (float64(hi.Rect.X) - vx) * zoom
				y := (float64(hi.Rect.Y) - vy) * zoom
				w := float64(hi.Rect.Width) * zoom
				h := float64(hi.Rect.Height) * zoom
				col := color.RGBA{R: 0xff, A: 0xff}
				ebitenutil.DrawLine(world, x, y, x+w, y, col)
				ebitenutil.DrawLine(world, x+w, y, x+w, y+h, col)
				ebitenutil.DrawLine(world, x+w, y+h, x, y+h, col)
				ebitenutil.DrawLine(world, x, y+h, x, y, col)
			}

			// draw recent collision highlights (from resolver)
			recents := component.GetRecentHighlights()
			if len(recents) > 0 {
				for _, r := range recents {
					x := (float64(r.Hit.X) - vx) * zoom
					y := (float64(r.Hit.Y) - vy) * zoom
					w := float64(r.Hit.Width) * zoom
					h := float64(r.Hit.Height) * zoom
					col := color.RGBA{R: 0xff, A: 0x88}
					ebitenutil.DrawRect(world, x, y, w, h, col)
					x = (float64(r.Hurt.X) - vx) * zoom
					y = (float64(r.Hurt.Y) - vy) * zoom
					w = float64(r.Hurt.Width) * zoom
					h = float64(r.Hurt.Height) * zoom
					col = color.RGBA{R: 0xff, A: 0x88}
					ebitenutil.DrawRect(world, x, y, w, h, col)
				}
			}
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
