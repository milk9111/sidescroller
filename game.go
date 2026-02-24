package main

import (
	"fmt"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/ecs/system"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/prefabs"
)

type Game struct {
	frames           int
	hitFreeze        int
	world            *ecs.World
	scheduler        *ecs.Scheduler
	render           *system.RenderSystem
	physics          *system.PhysicsSystem
	camera           *system.CameraSystem
	debugPhysics     bool
	debugOverlay     bool
	prefabWatcher    *prefabs.Watcher
	levelName        string
	initialLevelName string
	allAbilities     bool
}

func NewGame(levelName string, debug bool, allAbilities bool, watchPrefabs bool, overlay bool) *Game {
	physicsSystem := system.NewPhysicsSystem()
	game := &Game{
		world:            ecs.NewWorld(),
		scheduler:        ecs.NewScheduler(),
		render:           system.NewRenderSystem(),
		physics:          physicsSystem,
		debugPhysics:     debug,
		debugOverlay:     overlay,
		levelName:        levelName,
		initialLevelName: levelName,
		allAbilities:     allAbilities,
	}

	cameraSystem := system.NewCameraSystem()

	// Add systems in the order they should update
	game.scheduler.Add(system.NewInputSystem())
	game.scheduler.Add(system.NewAudioSystem())
	game.scheduler.Add(system.NewPlayerControllerSystem())
	game.scheduler.Add(system.NewPathfindingSystem())
	game.scheduler.Add(system.NewAINavigationSystem())
	game.scheduler.Add(system.NewAIPhaseSystem())
	game.scheduler.Add(system.NewCooldownSystem())
	game.scheduler.Add(system.NewAISystem())
	game.scheduler.Add(system.NewAimSystem())
	game.scheduler.Add(system.NewAnimationSystem())
	game.scheduler.Add(system.NewWhiteFlashSystem())
	game.scheduler.Add(system.NewInvulnerabilitySystem())
	game.scheduler.Add(system.NewDamageKnockbackSystem())
	game.scheduler.Add(system.NewCombatSystem())
	game.scheduler.Add(system.NewArenaNodeSystem())
	game.scheduler.Add(system.NewGateSystem())
	game.scheduler.Add(system.NewPlayerHealthBarSystem())
	game.scheduler.Add(system.NewHitFreezeSystem(game.setHitFreeze))
	game.scheduler.Add(system.NewHazardSystem())
	game.scheduler.Add(system.NewAnchorSystem())
	game.scheduler.Add(system.NewClusterRepulsionSystem())
	game.scheduler.Add(physicsSystem)
	game.scheduler.Add(system.NewPickupHoverSystem())
	game.scheduler.Add(system.NewPickupCollectSystem())
	game.scheduler.Add(system.NewTTLSystem())
	game.scheduler.Add(system.NewRespawnSystem())
	game.scheduler.Add(system.NewTransitionPopSystem())
	game.scheduler.Add(system.NewTransitionSystem())
	game.scheduler.Add(cameraSystem)

	game.camera = cameraSystem

	if err := game.reloadWorld(); err != nil {
		panic("failed to load world: " + err.Error())
	}

	if watchPrefabs {
		watcher, err := prefabs.NewWatcher("prefabs", "prefabs/scripts")
		if err != nil {
			panic("failed to create prefab watcher: " + err.Error())
		}

		game.prefabWatcher = watcher
	}

	return game
}

func (g *Game) Update() error {
	g.frames++

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		return ErrQuit
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF11) {
		g.levelName = g.initialLevelName
		return g.reloadWorld()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.debugPhysics = !g.debugPhysics
	}

	if g.hitFreeze > 0 {
		g.hitFreeze--
		return nil
	}

	g.scheduler.Update(g.world)

	if err := g.processPrefabEvents(); err != nil {
		panic("failed to process prefab events: " + err.Error())
	}

	// If any system requested a reload (e.g. player death finished), perform it now.
	if _, ok := ecs.First(g.world, component.ReloadRequestComponent.Kind()); ok {
		return g.reloadWorld()
	}

	// If any system requested a level change (TransitionSystem sent the
	// LevelChangeRequest after fade-out), perform reload and spawn now.
	if req, ok := g.firstLevelChangeRequest(); ok {
		// Remove the request entity so it can't be reprocessed.
		ecs.ForEach(g.world, component.LevelChangeRequestComponent.Kind(), func(e ecs.Entity, _ *component.LevelChangeRequest) {
			ecs.DestroyEntity(g.world, e)
		})

		if req.TargetLevel != "" {
			g.levelName = req.TargetLevel
		}
		// Debug: log request values to verify EnterDir/FromFacingLeft
		fmt.Printf("LevelChangeRequest: Target=%q SpawnTransitionID=%q EnterDir=%q FromFacingLeft=%v\n", req.TargetLevel, req.SpawnTransitionID, string(req.EnterDir), req.FromFacingLeft)

		// Preserve player health across the world reload performed for
		// level transitions. Capture current player's health (if any),
		// then reapply it to the newly spawned player after reload.
		var savedHealth *component.Health
		var savedAbilities *component.Abilities
		if p, ok := ecs.First(g.world, component.PlayerTagComponent.Kind()); ok {
			if h, ok := ecs.Get(g.world, p, component.HealthComponent.Kind()); ok && h != nil {
				savedHealth = &component.Health{Initial: h.Initial, Current: h.Current}
			}
		}
		if abEnt, ok := ecs.First(g.world, component.AbilitiesComponent.Kind()); ok {
			if ab, ok := ecs.Get(g.world, abEnt, component.AbilitiesComponent.Kind()); ok && ab != nil {
				savedAbilities = &component.Abilities{DoubleJump: ab.DoubleJump, WallGrab: ab.WallGrab, Anchor: ab.Anchor}
			}
		}

		if err := g.reloadWorld(); err != nil {
			return err
		}
		g.spawnPlayerAtLinkedTransition(req.SpawnTransitionID)

		// Reapply saved health to the new player entity so transitions do
		// not reset the player's health value.
		if savedHealth != nil {
			if newPlayer, ok := ecs.First(g.world, component.PlayerTagComponent.Kind()); ok {
				_ = ecs.Add(g.world, newPlayer, component.HealthComponent.Kind(), savedHealth)
			}
		}
		if savedAbilities != nil {
			if abEnt, ok := ecs.First(g.world, component.AbilitiesComponent.Kind()); ok {
				_ = ecs.Add(g.world, abEnt, component.AbilitiesComponent.Kind(), savedAbilities)
			}
		}

		// Run one scheduler tick so systems can initialize in the new world.
		if g.scheduler != nil && g.world != nil {
			g.scheduler.Update(g.world)
		}
		// After the scheduler tick physics bodies should exist; add a
		// one-shot `TransitionPop` component to the player which the
		// `TransitionPopSystem` will process (runs after physics).
		// Only apply for upward spawn transitions when the player entered
		// from below.
		if req.EntryFromBelow {
			// Find the spawn transition in the new world to inspect its EnterDir.
			spawnEnterDir := component.TransitionDirection("")
			ecs.ForEach2(g.world, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(ent ecs.Entity, tr *component.Transition, _ *component.Transform) {
				if tr != nil && tr.ID == req.SpawnTransitionID {
					spawnEnterDir = tr.EnterDir
				}
			})
			if spawnEnterDir == component.TransitionDirUp {
				if player, ok := ecs.First(g.world, component.PlayerTagComponent.Kind()); ok {
					mv := 80.0
					jp := 120.0
					if pCfg, ok := ecs.Get(g.world, player, component.PlayerComponent.Kind()); ok && pCfg != nil {
						mv = pCfg.MoveSpeed
						jp = pCfg.JumpSpeed
					}
					side := 1.0
					if req.FromFacingLeft {
						side = -1.0
					}

					dur := 6
					push := mv * 8.0

					pop := &component.TransitionPop{
						VX:          side * mv * 0.75,
						VY:          -jp * 1.1,
						FacingLeft:  req.FromFacingLeft,
						WallJumpDur: dur,
						WallJumpX:   side * push,
					}
					_ = ecs.Add(g.world, player, component.TransitionPopComponent.Kind(), pop)
				}
			}
		}
		// Signal to systems that load/spawn has completed.
		ent := ecs.CreateEntity(g.world)
		_ = ecs.Add(g.world, ent, component.LevelLoadedComponent.Kind(), &component.LevelLoaded{})

		// Create a TransitionRuntime in the new world so the fade-in can be
		// performed by the TransitionSystem running on the current world.
		rtEnt := ecs.CreateEntity(g.world)
		_ = ecs.Add(g.world, rtEnt, component.TransitionRuntimeComponent.Kind(), &component.TransitionRuntime{
			Phase:   component.TransitionFadeIn,
			Alpha:   1,
			Timer:   30,
			Req:     component.LevelChangeRequest{},
			ReqSent: true,
		})
	}

	return nil
}

func (g *Game) setHitFreeze(frames int) {
	if g == nil || frames <= 0 {
		return
	}
	if frames > g.hitFreeze {
		g.hitFreeze = frames
	}
}

func (g *Game) firstLevelChangeRequest() (component.LevelChangeRequest, bool) {
	if g == nil || g.world == nil {
		return component.LevelChangeRequest{}, false
	}
	ent, ok := ecs.First(g.world, component.LevelChangeRequestComponent.Kind())
	if !ok {
		return component.LevelChangeRequest{}, false
	}
	req, ok := ecs.Get(g.world, ent, component.LevelChangeRequestComponent.Kind())
	return *req, ok
}

// Transition timing/state is now managed by the TransitionSystem.

func (g *Game) Draw(screen *ebiten.Image) {
	if g.render != nil {
		g.render.Draw(g.world, screen)
	}
	if g.debugPhysics && g.physics != nil {
		system.DrawPhysicsDebug(g.physics.Space(), g.world, screen)
		system.DrawAIStateDebug(g.world, screen)
		system.DrawPathfindingDebug(g.world, screen)
		// Draw hazard component debug overlays
		system.DrawHazardDebug(g.world, screen)
	}

	// Player state debug text overlay (optional)
	if g.debugOverlay {
		system.DrawPlayerStateDebug(g.world, screen)
	}
}

func (g *Game) spawnPlayerAtLinkedTransition(transitionID string) {
	if g == nil || g.world == nil || transitionID == "" {
		return
	}

	player, ok := ecs.First(g.world, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	var (
		spawnX  float64
		spawnY  float64
		spawnH  float64
		found   bool
		isLeft  bool
		isRight bool
	)

	ecs.ForEach2(g.world, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(ent ecs.Entity, tr *component.Transition, tf *component.Transform) {
		if found || tr.ID != transitionID {
			return
		}

		w := tr.Bounds.W
		h := tr.Bounds.H

		if w <= 0 {
			w = 32
		}
		if h <= 0 {
			h = 32

		}
		spawnX = tf.X + tr.Bounds.X + w/2
		spawnY = tf.Y + tr.Bounds.Y + h/2
		spawnH = h
		found = true
		isLeft = tr.EnterDir == component.TransitionDirLeft
		isRight = tr.EnterDir == component.TransitionDirRight
	})

	if !found {
		return
	}

	playerTf, ok := ecs.Get(g.world, player, component.TransformComponent.Kind())
	if !ok {
		playerTf = &component.Transform{ScaleX: 1, ScaleY: 1}
	}
	playerBody, ok := ecs.Get(g.world, player, component.PhysicsBodyComponent.Kind())
	if ok && playerBody.Width > 0 && playerBody.Height > 0 {
		if !isLeft && !isRight {
			playerTf.X = spawnX - playerBody.Width/2 - playerBody.OffsetX
			playerTf.Y = spawnY - playerBody.Height/2 - playerBody.OffsetY
		} else {
			playerTf.X = spawnX - playerBody.Width/2 - playerBody.OffsetX
			playerTf.Y = spawnY + spawnH/2 - playerBody.Height
		}
	} else {
		// Fallback: treat transform as top-left.
		playerTf.X = spawnX
		playerTf.Y = spawnY
	}

	playerSprite, ok := ecs.Get(g.world, player, component.SpriteComponent.Kind())
	if ok {
		playerSprite.FacingLeft = isLeft
	}

	// Lock out immediate re-trigger until the player leaves the spawn transition.
	_ = ecs.Add(g.world, player, component.TransitionCooldownComponent.Kind(), &component.TransitionCooldown{Active: true, TransitionID: transitionID})
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	if g.camera != nil {
		g.camera.SetScreenSize(outsideWidth, outsideHeight)
	}

	return outsideWidth, outsideHeight
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	panic("shouldn't use Layout")
}

func (g *Game) reloadWorld() error {
	// Reset physics system state to avoid retaining bodies/shapes from the
	// previous world which can cause entities to appear at old positions.
	if g.physics != nil {
		g.physics.Reset()
	}

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

	if _, err = entity.BuildEntity(world, "camera.yaml"); err != nil {
		return err
	}

	if len(level.Entities) == 0 {
		if _, err = entity.BuildEntity(world, "player.yaml"); err != nil {
			return err
		}
	}

	if _, err = entity.BuildEntity(world, "aim_target.yaml"); err != nil {
		return err
	}

	if _, err = entity.NewPlayerHealthBar(world); err != nil {
		return err
	}

	if _, err = entity.BuildEntity(world, "ambience.yaml"); err != nil {
		return err
	}

	g.world = world
	// Add Abilities component to the world so systems can query which
	// optional abilities are enabled. Defaults to only basic jump when
	// `allAbilities` is false.
	abEnt := ecs.CreateEntity(g.world)
	_ = ecs.Add(g.world, abEnt, component.AbilitiesComponent.Kind(), &component.Abilities{
		DoubleJump: g.allAbilities,
		WallGrab:   g.allAbilities,
		Anchor:     g.allAbilities,
	})
	// Signal to systems that the level has finished loading so the camera
	// and other systems can perform any immediate setup (e.g. snap camera).
	ent := ecs.CreateEntity(g.world)
	_ = ecs.Add(g.world, ent, component.LevelLoadedComponent.Kind(), &component.LevelLoaded{})
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
