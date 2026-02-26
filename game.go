package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/system"
	"github.com/milk9111/sidescroller/prefabs"
)

type Game struct {
	frames        int
	hitFreeze     int
	world         *ecs.World
	scheduler     *ecs.Scheduler
	persistence   *system.PersistenceSystem
	render        *system.RenderSystem
	physics       *system.PhysicsSystem
	camera        *system.CameraSystem
	debugPhysics  bool
	debugOverlay  bool
	prefabWatcher *prefabs.Watcher
}

func NewGame(levelName string, debug bool, allAbilities bool, watchPrefabs bool, overlay bool) *Game {
	physicsSystem := system.NewPhysicsSystem()
	persistenceSystem := system.NewPersistenceSystem(levelName, allAbilities, physicsSystem.Reset)
	game := &Game{
		world:        ecs.NewWorld(),
		scheduler:    ecs.NewScheduler(),
		persistence:  persistenceSystem,
		render:       system.NewRenderSystem(),
		physics:      physicsSystem,
		debugPhysics: debug,
		debugOverlay: overlay,
	}

	cameraSystem := system.NewCameraSystem()

	// Add systems in the order they should update
	game.scheduler.Add(system.NewInputSystem())
	game.scheduler.Add(system.NewAudioSystem())
	game.scheduler.Add(system.NewMusicSystem())
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
	game.scheduler.Add(system.NewTrophyCounterSystem())
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
	game.scheduler.Add(game.persistence)
	game.scheduler.Add(cameraSystem)

	game.camera = cameraSystem

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

func (g *Game) Draw(screen *ebiten.Image) {
	if g.render != nil {
		g.render.Draw(g.world, screen)
	}
	if g.debugPhysics && g.physics != nil {
		system.DrawPhysicsDebug(g.physics.Space(), g.world, screen)
		system.DrawAIStateDebug(g.world, screen)
		system.DrawPathfindingDebug(g.world, screen)
		system.DrawPickupDebug(g.world, screen)
		system.DrawTransitionDebug(g.world, screen)
		system.DrawHazardDebug(g.world, screen)
	}

	// Player state debug text overlay (optional)
	if g.debugOverlay {
		system.DrawPlayerStateDebug(g.world, screen)
	}
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
				ent := ecs.CreateEntity(g.world)
				_ = ecs.Add(g.world, ent, component.ReloadRequestComponent.Kind(), &component.ReloadRequest{})
				return nil
			}
			return nil
		}
	}
}
