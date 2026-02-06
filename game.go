package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"strings"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
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
	pickups        []*obj.Pickup
	enemies        []*obj.Enemy
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

	paused bool
}

func NewGame(levelPath string, debug bool, allAbilities bool) *Game {
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
	if lvl != nil && len(lvl.Entities) > 0 {
		remaining := lvl.Entities
		g.pickups, remaining = g.spawnPickupsFromEntities(remaining)
		g.enemies, remaining = g.spawnEnemiesFromEntities(remaining, g.collisionWorld)
		g.level.Entities = remaining
	}

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
			// spawn pickups/enemies from placed entities and remove them from level.Entities
			g.pickups = nil
			g.enemies = nil
			if g.level != nil && len(g.level.Entities) > 0 {
				remaining := g.level.Entities
				g.pickups, remaining = g.spawnPickupsFromEntities(remaining)
				g.enemies, remaining = g.spawnEnemiesFromEntities(remaining, g.collisionWorld)
				g.level.Entities = remaining
			}
			g.player = obj.NewPlayer(spawnX, spawnY, g.input, g.collisionWorld, g.anchor, g.player.IsFacingRight(), g.player.DoubleJumpEnabled, g.player.WallGrabEnabled, g.player.SwingEnabled, g.player.DashEnabled)
			// Update the input reference to the newly created player so gamepad
			// aiming continues to target the correct player after a level change.
			g.input.Player = g.player
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

func (g *Game) spawnPickupsFromEntities(entities []obj.PlacedEntity) ([]*obj.Pickup, []obj.PlacedEntity) {
	if g == nil || len(entities) == 0 {
		return nil, entities
	}

	pickups := make([]*obj.Pickup, 0)
	remaining := make([]obj.PlacedEntity, 0, len(entities))
	for _, pe := range entities {
		if !isPickupEntity(pe) {
			remaining = append(remaining, pe)
			continue
		}

		x := float32(pe.X * common.TileSize)
		y := float32(pe.Y * common.TileSize)

		var pickup *obj.Pickup
		switch {
		case strings.Contains(pe.Name, "dash_pickup"):
			var p *obj.Pickup
			p = obj.NewPickup(x, y, pe.Sprite, func() {
				g.player.DashEnabled = true
				if p != nil {
					p.Disabled = true
				}
			})
			pickup = p
		case strings.Contains(pe.Name, "anchor_pickup"):
			var p *obj.Pickup
			p = obj.NewPickup(x, y, pe.Sprite, func() {
				g.player.SwingEnabled = true
				if p != nil {
					p.Disabled = true
				}
			})
			pickup = p
		case strings.Contains(pe.Name, "double_jump_pickup"):
			var p *obj.Pickup
			p = obj.NewPickup(x, y, pe.Sprite, func() {
				g.player.DoubleJumpEnabled = true
				if p != nil {
					p.Disabled = true
				}
			})
			pickup = p
		default:
			remaining = append(remaining, pe)
			continue
		}

		pickups = append(pickups, pickup)
	}

	return pickups, remaining
}

func (g *Game) spawnEnemiesFromEntities(entities []obj.PlacedEntity, collisionWorld *obj.CollisionWorld) ([]*obj.Enemy, []obj.PlacedEntity) {
	if g == nil || len(entities) == 0 {
		return nil, entities
	}

	enemies := make([]*obj.Enemy, 0)
	remaining := make([]obj.PlacedEntity, 0, len(entities))
	for _, pe := range entities {
		if !isEnemyEntity(pe) {
			remaining = append(remaining, pe)
			continue
		}

		x := float32(pe.X * common.TileSize)
		y := float32(pe.Y * common.TileSize)
		enemy := obj.NewEnemy(x, y, collisionWorld)
		if enemy != nil {
			enemies = append(enemies, enemy)
		}
	}

	return enemies, remaining
}

func isPickupEntity(pe obj.PlacedEntity) bool {
	if strings.EqualFold(strings.TrimSpace(pe.Type), "pickup") {
		return true
	}
	return pe.Type == "" && strings.Contains(pe.Name, "pickup")
}

func isEnemyEntity(pe obj.PlacedEntity) bool {
	if strings.EqualFold(strings.TrimSpace(pe.Type), "enemy") {
		return true
	}
	return pe.Type == "" && strings.Contains(pe.Name, "enemy")
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

	resolver := component.NewCombatResolver()
	// Add a resolver-level emitter to trigger camera shake on damage to player
	if g.camera != nil {
		em := component.CombatEventEmitter{}
		em.Handlers = append(em.Handlers, func(evt component.CombatEvent) {
			if evt.Type == component.EventDamageApplied && g.player != nil && evt.TargetID == g.player.ID {
				g.camera.StartShake(6.0, 12)
			}
		})
		resolver.Emitter = &em
	}
	// collect dealers (players + enemies)
	dealers := make([]component.DamageDealerComponent, 0)
	targets := make([]component.HurtboxComponent, 0)
	healthByOwner := make(map[int]component.HealthComponent)
	if g.player != nil {
		dealers = append(dealers, g.player)
		targets = append(targets, g.player)
		if g.player.Health() != nil {
			h := g.player.Health()
			healthByOwner[g.player.ID] = h
			// ensure camera shake handler is attached to the live Health instance
			if g.camera != nil {
				h.OnDamage = func(hh *component.Health, evt component.CombatEvent) {
					log.Printf("Player took damage (handler): amt=%.2f attacker=%d target=%d frame=%d", evt.Damage, evt.AttackerID, evt.TargetID, evt.Frame)
					g.camera.StartShake(6.0, 12)
				}
			}
		}
	}
	for _, enemy := range g.enemies {
		if enemy == nil {
			continue
		}
		dealers = append(dealers, enemy)
		targets = append(targets, enemy)
	}
	if len(dealers) > 0 && len(targets) > 0 {
		resolver.Tick()
		resolver.ResolveAll(dealers, targets, healthByOwner)
		if len(resolver.Recent) > 0 {
			component.AddRecentHighlights(resolver.Recent)
		}
	}

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
