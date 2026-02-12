package main

import (
	"os"
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
	frames        int
	world         *ecs.World
	scheduler     *ecs.Scheduler
	render        *system.RenderSystem
	physics       *system.PhysicsSystem
	camera        *system.CameraSystem
	debugPhysics  bool
	prefabWatcher *prefabs.Watcher
	levelName     string
}

func NewGame(levelName string, debug bool, allAbilities bool) *Game {
	physicsSystem := system.NewPhysicsSystem()
	game := &Game{
		world:        ecs.NewWorld(),
		scheduler:    ecs.NewScheduler(),
		render:       system.NewRenderSystem(),
		physics:      physicsSystem,
		debugPhysics: debug,
		levelName:    levelName,
	}

	cameraSystem := system.NewCameraSystem()

	// Add systems in the order they should update
	game.scheduler.Add(system.NewInputSystem())
	game.scheduler.Add(system.NewPlayerControllerSystem())
	game.scheduler.Add(system.NewPathfindingSystem())
	game.scheduler.Add(system.NewAISystem())
	game.scheduler.Add(system.NewAimSystem())
	game.scheduler.Add(system.NewAnimationSystem())
	game.scheduler.Add(system.NewWhiteFlashSystem())
	game.scheduler.Add(system.NewCombatSystem())
	game.scheduler.Add(physicsSystem)
	// Transition checks should run after physics has synced transforms.
	game.scheduler.Add(system.NewTransitionSystem())
	game.scheduler.Add(system.NewAnchorSystem())
	game.scheduler.Add(cameraSystem)

	game.camera = cameraSystem

	if err := game.reloadWorld(); err != nil {
		panic("failed to load world: " + err.Error())
	}

	watcher, err := prefabs.NewWatcher("prefabs")
	if err != nil {
		panic("failed to create prefab watcher: " + err.Error())
	}

	game.prefabWatcher = watcher

	return game
}

func (g *Game) Update() error {
	g.frames++

	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		os.Exit(0)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.debugPhysics = !g.debugPhysics
	}

	g.scheduler.Update(g.world)

	if err := g.processPrefabEvents(); err != nil {
		panic("failed to process prefab events: " + err.Error())
	}

	// If any system requested a reload (e.g. player death finished), perform it now.
	if _, ok := g.world.First(component.ReloadRequestComponent.Kind()); ok {
		return g.reloadWorld()
	}

	// If any system requested a level change (TransitionSystem sent the
	// LevelChangeRequest after fade-out), perform reload and spawn now.
	if req, ok := g.firstLevelChangeRequest(); ok {
		// Remove the request entity so it can't be reprocessed.
		for _, e := range g.world.Query(component.LevelChangeRequestComponent.Kind()) {
			g.world.DestroyEntity(e)
		}
		if req.TargetLevel != "" {
			g.levelName = req.TargetLevel
		}
		if err := g.reloadWorld(); err != nil {
			return err
		}
		g.spawnPlayerAtLinkedTransition(req.SpawnTransitionID)
		// Run one scheduler tick so systems can initialize in the new world.
		if g.scheduler != nil && g.world != nil {
			g.scheduler.Update(g.world)
		}
		// Signal to systems that load/spawn has completed.
		ent := g.world.CreateEntity()
		_ = ecs.Add(g.world, ent, component.LevelLoadedComponent, component.LevelLoaded{})

		// Create a TransitionRuntime in the new world so the fade-in can be
		// performed by the TransitionSystem running on the current world.
		rtEnt := g.world.CreateEntity()
		_ = ecs.Add(g.world, rtEnt, component.TransitionRuntimeComponent, component.TransitionRuntime{
			Phase:   component.TransitionFadeIn,
			Alpha:   1,
			Timer:   30,
			Req:     component.LevelChangeRequest{},
			ReqSent: true,
		})
	}

	return nil
}

func (g *Game) firstLevelChangeRequest() (component.LevelChangeRequest, bool) {
	if g == nil || g.world == nil {
		return component.LevelChangeRequest{}, false
	}
	ent, ok := g.world.First(component.LevelChangeRequestComponent.Kind())
	if !ok {
		return component.LevelChangeRequest{}, false
	}
	req, ok := ecs.Get(g.world, ent, component.LevelChangeRequestComponent)
	return req, ok
}

// Transition timing/state is now managed by the TransitionSystem.

func (g *Game) Draw(screen *ebiten.Image) {
	if g.render != nil {
		g.render.Draw(g.world, screen)
	}
	if g.debugPhysics && g.physics != nil {
		system.DrawPhysicsDebug(g.physics.Space(), g.world, screen)
		system.DrawPlayerStateDebug(g.world, screen)
		system.DrawAIStateDebug(g.world, screen)
		system.DrawPathfindingDebug(g.world, screen)
	}
}

func (g *Game) spawnPlayerAtLinkedTransition(transitionID string) {
	if g == nil || g.world == nil || transitionID == "" {
		return
	}

	player, ok := g.world.First(component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	var (
		spawnX float64
		spawnY float64
		found  bool
	)
	for _, e := range g.world.Query(component.TransitionComponent.Kind(), component.TransformComponent.Kind()) {
		tr, ok := ecs.Get(g.world, e, component.TransitionComponent)
		if !ok || tr.ID != transitionID {
			continue
		}
		tf, _ := ecs.Get(g.world, e, component.TransformComponent)
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
		found = true
		break
	}
	if !found {
		return
	}

	playerTf, ok := ecs.Get(g.world, player, component.TransformComponent)
	if !ok {
		playerTf = component.Transform{ScaleX: 1, ScaleY: 1}
	}
	playerBody, ok := ecs.Get(g.world, player, component.PhysicsBodyComponent)
	if ok && playerBody.Width > 0 && playerBody.Height > 0 {
		playerTf.X = spawnX - playerBody.Width/2 - playerBody.OffsetX
		playerTf.Y = spawnY - playerBody.Height/2 - playerBody.OffsetY
	} else {
		// Fallback: treat transform as top-left.
		playerTf.X = spawnX
		playerTf.Y = spawnY
	}
	_ = ecs.Add(g.world, player, component.TransformComponent, playerTf)

	// Lock out immediate re-trigger until the player leaves the spawn transition.
	_ = ecs.Add(g.world, player, component.TransitionCooldownComponent, component.TransitionCooldown{Active: true, TransitionID: transitionID})
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

	if _, err = entity.NewCamera(world); err != nil {
		return err
	}

	if len(level.Entities) == 0 {
		if _, err = entity.NewPlayer(world); err != nil {
			return err
		}
	}

	if _, err = entity.NewAimTarget(world); err != nil {
		return err
	}

	g.world = world
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
