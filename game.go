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
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/ecs/systems"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/obj"

	"github.com/ebitenui/ebitenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	frames int

	input  *obj.Input
	level  *obj.Level
	camera *obj.Camera

	debugDraw bool
	baseZoom  float64
	// recentlyTeleported prevents immediate retriggering of transitions
	recentlyTeleported bool
	// Transition manager (handles fade/load/fade)
	transition   *obj.Transition
	ui           *ebitenui.UI
	ecsWorld     *ecs.World
	ecsPlayer    ecs.Entity
	ecsSpawn     *systems.SpawnSystem
	ecsAI        *systems.AISystem
	ecsInput     *systems.InputSystem
	ecsCamera    ecs.Entity
	ecsCameraSys *systems.CameraSystem

	abilityDoubleJump bool
	abilityWallGrab   bool
	abilitySwing      bool
	abilityDash       bool
	playerHealth      *components.Health

	paused bool
}

func NewGame(levelPath string, debug bool, allAbilities bool) *Game {
	lvl, err := loadLevel(levelPath)
	if err != nil {
		log.Printf("failed to load level %s: %v", levelPath, err)
	}
	if lvl == nil {
		lvl = &obj.Level{Width: 1, Height: 1}
	}

	levelW := lvl.Width * common.TileSize
	levelH := lvl.Height * common.TileSize
	baseZoom := 2.0
	camera := obj.NewCamera(common.BaseWidth, common.BaseHeight, baseZoom)
	camera.SetWorldBounds(levelW, levelH)

	input := obj.NewInput(camera)

	g := &Game{
		input:             input,
		level:             lvl,
		debugDraw:         debug,
		camera:            camera,
		baseZoom:          baseZoom,
		transition:        obj.NewTransition(),
		abilityDoubleJump: allAbilities,
		abilityWallGrab:   allAbilities,
		abilitySwing:      allAbilities,
		abilityDash:       allAbilities,
	}

	spawnX, spawnY := lvl.GetSpawnPosition()
	if g.camera != nil {
		cx := float64(spawnX) + 8
		cy := float64(spawnY) + 20
		g.camera.SnapTo(cx, cy)
	}
	g.initECS(spawnX, spawnY, lvl, input, camera, false)

	// create pause UI
	g.ui = NewPauseUI(g)

	// wire transition callback to perform the actual level load and setup
	if g.transition != nil {
		g.transition.OnStart = func(target, linkID, direction string) {
			g.capturePlayerState()
			lvl, err := loadLevel(target)
			if err != nil {
				log.Printf("failed to load transition target %s: %v", target, err)
				return
			}
			if lvl == nil {
				return
			}
			g.level = lvl
			applyJump := strings.ToLower(direction) == "up"
			spawnX, spawnY := resolveTransitionSpawn(lvl, linkID, direction)
			if g.camera != nil {
				levelW := lvl.Width * common.TileSize
				levelH := lvl.Height * common.TileSize
				g.camera.SetWorldBounds(levelW, levelH)
				g.camera.SnapTo(float64(spawnX)+8, float64(spawnY)+20)
			}
			g.initECS(spawnX, spawnY, lvl, g.input, g.camera, applyJump)
			g.recentlyTeleported = true
		}
	}

	return g
}

func (g *Game) initECS(spawnX, spawnY float32, level *obj.Level, input *obj.Input, camera *obj.Camera, applyJump bool) {
	if g == nil || level == nil {
		return
	}
	combatEvents := make([]component.CombatEvent, 0, 32)
	g.ecsWorld = ecs.NewWorld()
	g.ecsPlayer = g.ecsWorld.CreateEntity()

	g.ecsWorld.SetTransform(g.ecsPlayer, &components.Transform{X: spawnX, Y: spawnY})
	g.ecsWorld.SetVelocity(g.ecsPlayer, &components.Velocity{})
	g.ecsWorld.SetGravity(g.ecsPlayer, &components.Gravity{Enabled: true})
	g.ecsWorld.SetCollider(g.ecsPlayer, &components.Collider{Width: 16, Height: 40, FixedRotation: true, IsPlayer: true})
	g.ecsWorld.SetGroundSensor(g.ecsPlayer, &components.GroundSensor{})
	g.ecsWorld.SetCollisionState(g.ecsPlayer, &components.CollisionState{})
	g.ecsWorld.SetInput(g.ecsPlayer, &components.InputState{})

	if g.playerHealth == nil {
		g.playerHealth = &components.Health{Current: 5, Max: 5}
	}
	ecsHealth := &components.Health{
		Current: g.playerHealth.Current,
		Max:     g.playerHealth.Max,
		IFrames: g.playerHealth.IFrames,
		Dead:    g.playerHealth.Dead,
	}
	if camera != nil {
		ecsHealth.OnDamage = func(hh *components.Health, evt component.CombatEvent) {
			camera.StartShake(6.0, 12)
		}
	}
	g.ecsWorld.SetHealth(g.ecsPlayer, ecsHealth)
	g.playerHealth = ecsHealth

	animIdle := component.NewAnimationRow(assets.PlayerV2Sheet, 64, 64, 0, 8, 12, true)
	animRun := component.NewAnimationRow(assets.PlayerV2Sheet, 64, 64, 1, 4, 12, true)
	animAttack := component.NewAnimationRow(assets.PlayerV2Sheet, 64, 64, 3, 5, 12, false)

	g.ecsWorld.SetSprite(g.ecsPlayer, &components.Sprite{
		Width:   64,
		Height:  64,
		OffsetX: -24,
		OffsetY: -20,
		Layer:   2,
	})
	g.ecsWorld.SetAnimator(g.ecsPlayer, &components.Animator{Anim: animIdle, Playing: true})

	ctrl := &components.PlayerController{
		MoveSpeed:    4.0,
		JumpVelocity: -8.0,
		MaxSpeedX:    6.0,
		FacingRight:  true,
		State:        "idle",
		AttackFrames: 2,
		CoyoteFrames: 6,
		MaxJumps:     1,
		DoubleJump:   g.abilityDoubleJump,
		WallGrab:     g.abilityWallGrab,
		Swing:        g.abilitySwing,
		Dash:         g.abilityDash,
		IdleAnim:     animIdle,
		RunAnim:      animRun,
		AttackAnim:   animAttack,
	}
	if ctrl.DoubleJump {
		ctrl.MaxJumps = 2
	}
	g.ecsWorld.SetPlayerController(g.ecsPlayer, ctrl)

	if applyJump {
		if vel := g.ecsWorld.GetVelocity(g.ecsPlayer); vel != nil {
			vel.VY = ctrl.JumpVelocity
		}
	}

	g.ecsCamera = g.ecsWorld.CreateEntity()
	g.ecsWorld.SetCameraFollow(g.ecsCamera, &components.CameraFollow{TargetEntity: g.ecsPlayer.ID})
	if camera != nil {
		g.ecsWorld.SetCameraState(g.ecsCamera, &components.CameraState{PosX: camera.PosX, PosY: camera.PosY, Zoom: camera.Zoom()})
	}
	g.ecsCameraSys = systems.NewCameraSystem(camera, g.ecsCamera)

	g.ecsSpawn = systems.NewSpawnSystem(level, g.ecsPlayer)
	g.ecsAI = systems.NewAISystem(level)
	g.ecsInput = systems.NewInputSystem(input, g.ecsPlayer)
	g.ecsWorld.SetPhysicsWorld(ecs.NewPhysicsWorld(level))

	g.ecsWorld.AddSystem(g.ecsInput)
	g.ecsWorld.AddSystem(systems.NewPlayerControllerSystem())
	g.ecsWorld.AddSystem(systems.NewPlayerCombatSystem())
	g.ecsWorld.AddSystem(systems.NewPickupSystem())
	g.ecsWorld.AddSystem(g.ecsSpawn)
	g.ecsWorld.AddSystem(g.ecsAI)
	g.ecsWorld.AddSystem(systems.NewMovementSystem())
	g.ecsWorld.AddSystem(systems.NewCollisionSystem())
	g.ecsWorld.AddSystem(systems.NewBulletSystem(level))
	g.ecsWorld.AddSystem(systems.NewHealthSystem())
	g.ecsWorld.AddSystem(systems.NewHurtboxSyncSystem())
	g.ecsWorld.AddSystem(systems.NewCombatSystem(&combatEvents))
	g.ecsWorld.AddSystem(systems.NewDamageSystem(&combatEvents))
	g.ecsWorld.AddSystem(systems.NewBulletCleanupSystem(&combatEvents))
	g.ecsWorld.AddSystem(systems.NewAnimationSystem(nil))
	g.ecsWorld.AddSystem(systems.NewRenderSystem())
	g.ecsWorld.AddSystem(g.ecsCameraSys)
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

	g.input.Update()

	if g.ecsWorld != nil {
		g.ecsWorld.Update()
		g.capturePlayerState()
	}

	// tick global highlight store
	component.TickHighlights()

	// handle level transitions: if player overlaps a transition rect, load target level
	if g.level != nil && g.ecsWorld != nil {
		tr := g.ecsWorld.GetTransform(g.ecsPlayer)
		col := g.ecsWorld.GetCollider(g.ecsPlayer)
		if tr != nil && col != nil {
			px := tr.X
			py := tr.Y
			pw := col.Width
			ph := col.Height
			left := int(math.Floor(float64(px) / float64(common.TileSize)))
			top := int(math.Floor(float64(py) / float64(common.TileSize)))
			right := int(math.Floor(float64(px+pw-1) / float64(common.TileSize)))
			bottom := int(math.Floor(float64(py+ph-1) / float64(common.TileSize)))

			overlapping := false
			var hitTr *obj.TransitionData
			for i := range g.level.Transitions {
				trn := &g.level.Transitions[i]
				if right < trn.X || left > trn.X+trn.W-1 || bottom < trn.Y || top > trn.Y+trn.H-1 {
					continue
				}
				overlapping = true
				hitTr = trn
				break
			}

			if overlapping {
				if !g.recentlyTeleported && hitTr != nil && hitTr.Target != "" {
					if g.transition != nil {
						g.transition.Enter(hitTr.Target, hitTr.LinkID, hitTr.Direction)
					}
				}
			} else {
				g.recentlyTeleported = false
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	state := "unknown"
	gravityEnabled := false
	if g.ecsWorld != nil {
		if ctrl := g.ecsWorld.GetPlayerController(g.ecsPlayer); ctrl != nil {
			state = ctrl.State
		}
		if grav := g.ecsWorld.GetGravity(g.ecsPlayer); grav != nil {
			gravityEnabled = grav.Enabled
		}
	}
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Frames: %d    FPS: %.2f    State: %s    GravityEnabled: %t", g.frames, ebiten.ActualFPS(), state, gravityEnabled))
	g.camera.Render(screen, func(world *ebiten.Image) {
		vx, vy := g.camera.ViewTopLeft()
		zoom := g.camera.Zoom()
		g.level.Draw(world, vx, vy, zoom)
		if g.ecsWorld != nil {
			g.ecsWorld.Draw(world, vx, vy, zoom)
		}
		if g.debugDraw && g.ecsWorld != nil {
			if pw := g.ecsWorld.PhysicsWorld(); pw != nil {
				pw.DebugDraw(world, vx, vy, zoom)
			}
		}
		if g.debugDraw && g.ecsWorld != nil {
			if pathSet := g.ecsWorld.Pathfindings(); pathSet != nil {
				pathCol := color.RGBA{R: 0x40, G: 0xff, B: 0xff, A: 0xff}
				wayCol := color.RGBA{R: 0xff, G: 0xff, B: 0x40, A: 0xff}
				for _, id := range pathSet.Entities() {
					pv := pathSet.Get(id)
					pf, ok := pv.(*components.Pathfinding)
					if !ok || pf == nil || len(pf.Path) == 0 {
						continue
					}
					for i := 0; i < len(pf.Path)-1; i++ {
						a := pf.Path[i]
						b := pf.Path[i+1]
						ax := float64(a.X*common.TileSize+common.TileSize/2) - vx
						ay := float64(a.Y*common.TileSize+common.TileSize/2) - vy
						bx := float64(b.X*common.TileSize+common.TileSize/2) - vx
						by := float64(b.Y*common.TileSize+common.TileSize/2) - vy
						ebitenutil.DrawLine(world, ax*zoom, ay*zoom, bx*zoom, by*zoom, pathCol)
					}
					idx := pf.PathIndex
					if idx < 0 {
						idx = 0
					}
					if idx >= len(pf.Path) {
						idx = len(pf.Path) - 1
					}
					wp := pf.Path[idx]
					x := (float64(wp.X*common.TileSize) - vx) * zoom
					y := (float64(wp.Y*common.TileSize) - vy) * zoom
					sz := float64(common.TileSize) * zoom
					ebitenutil.DrawRect(world, x, y, sz, sz, wayCol)
				}
			}
		}

		if g.debugDraw && g.ecsWorld != nil {
			type boxRef struct {
				Owner  int
				ID     string
				Rect   common.Rect
				Active bool
				IsHit  bool
			}
			hitboxes := make([]boxRef, 0)
			hurtboxes := make([]boxRef, 0)

			if dealers := g.ecsWorld.DamageDealers(); dealers != nil {
				for _, id := range dealers.Entities() {
					if dv := dealers.Get(id); dv != nil {
						if d, ok := dv.(*components.DamageDealer); ok && d != nil {
							for _, hb := range d.Boxes {
								if !hb.Active {
									continue
								}
								hitboxes = append(hitboxes, boxRef{Owner: hb.OwnerID, ID: hb.ID, Rect: hb.Rect, Active: hb.Active, IsHit: true})
							}
						}
					}
				}
			}
			if hurtSet := g.ecsWorld.Hurtboxes(); hurtSet != nil {
				for _, id := range hurtSet.Entities() {
					if hv := hurtSet.Get(id); hv != nil {
						if h, ok := hv.(*components.HurtboxSet); ok && h != nil {
							for _, hu := range h.Boxes {
								if !hu.Enabled {
									continue
								}
								hurtboxes = append(hurtboxes, boxRef{Owner: hu.OwnerID, ID: hu.ID, Rect: hu.Rect, Active: hu.Enabled, IsHit: false})
							}
						}
					}
				}
			}

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

	// draw transition overlay (if active)
	if g.transition != nil {
		g.transition.Draw(screen)
	}

	// draw pause UI on top
	if g.paused && g.ui != nil {
		g.ui.Draw(screen)
	}
}

func (g *Game) capturePlayerState() {
	if g == nil || g.ecsWorld == nil {
		return
	}
	if ctrl := g.ecsWorld.GetPlayerController(g.ecsPlayer); ctrl != nil {
		g.abilityDoubleJump = ctrl.DoubleJump
		g.abilityWallGrab = ctrl.WallGrab
		g.abilitySwing = ctrl.Swing
		g.abilityDash = ctrl.Dash
	}
	if h := g.ecsWorld.GetHealth(g.ecsPlayer); h != nil {
		if g.playerHealth == nil {
			g.playerHealth = &components.Health{}
		}
		g.playerHealth.Current = h.Current
		g.playerHealth.Max = h.Max
		g.playerHealth.IFrames = h.IFrames
		g.playerHealth.Dead = h.Dead
	}
}

func loadLevel(levelPath string) (*obj.Level, error) {
	if levelPath == "" {
		return nil, fmt.Errorf("level path is empty")
	}
	if l, err := obj.LoadLevelFromFS(levels.LevelsFS, levelPath); err == nil {
		return l, nil
	}
	if l, err := obj.LoadLevel(levelPath); err == nil {
		return l, nil
	}
	return nil, fmt.Errorf("failed to load level %s", levelPath)
}

func resolveTransitionSpawn(level *obj.Level, linkID, direction string) (float32, float32) {
	if level == nil {
		return 0, 0
	}
	spawnX, spawnY := level.GetSpawnPosition()
	var targetTr *obj.TransitionData
	if linkID != "" {
		for i := range level.Transitions {
			tr := &level.Transitions[i]
			if tr.ID == linkID {
				spawnX = float32(tr.X * common.TileSize)
				spawnY = float32(tr.Y * common.TileSize)
				targetTr = tr
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
			centerX := float32(targetTr.X*common.TileSize) + float32(targetTr.W*common.TileSize)/2.0
			bottom := float32((targetTr.Y + targetTr.H) * common.TileSize)
			spawnX = centerX - 8.0
			spawnY = bottom - 40.0
		}
	}

	return spawnX, spawnY
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
