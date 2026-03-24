package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/system"
	"github.com/milk9111/sidescroller/prefabs"
)

type Game struct {
	frames        int
	world         *ecs.World
	gameplay      *ecs.Scheduler
	dialogue      *ecs.Scheduler
	active        *ecs.Scheduler
	dialogueInput *system.DialogueInputSystem
	dialogueOpen  bool
	pendingPress  bool
	input         *system.InputSystem
	ui            *system.UISystem
	persistence   *system.PersistenceSystem
	render        *system.RenderSystem
	physics       *system.PhysicsSystem
	camera        *system.CameraSystem
	scriptRuntime *system.ScriptSystem
	debugPhysics  bool
	debugOverlay  bool
	prefabWatcher *prefabs.Watcher
}

func NewGame(levelName string, debug bool, allAbilities bool, watchPrefabs bool, overlay bool, initialAbilities *component.Abilities) *Game {
	physicsSystem := system.NewPhysicsSystem()
	persistenceSystem := system.NewPersistenceSystem(levelName, allAbilities, initialAbilities, physicsSystem.Reset)
	inputSystem := system.NewInputSystem()
	uiSystem := system.NewUISystem()
	animationSystem := system.NewAnimationSystem()
	dialogueInputSystem := system.NewDialogueInputSystem()
	dialoguePopupSystem := system.NewDialoguePopupSystem()
	gameplayScheduler := ecs.NewScheduler()
	dialogueScheduler := ecs.NewScheduler()
	game := &Game{
		world:         ecs.NewWorld(),
		gameplay:      gameplayScheduler,
		dialogue:      dialogueScheduler,
		active:        gameplayScheduler,
		dialogueInput: dialogueInputSystem,
		input:         inputSystem,
		ui:            uiSystem,
		persistence:   persistenceSystem,
		render:        system.NewRenderSystem(),
		physics:       physicsSystem,
		debugPhysics:  debug,
		debugOverlay:  overlay,
	}

	cameraSystem := system.NewCameraSystem()
	scriptSystem := system.NewScriptSystem()
	musicSystem := system.NewMusicSystem()

	game.dialogue.Add(musicSystem)
	game.dialogue.Add(animationSystem)
	game.dialogue.Add(system.NewDialogueSystem())
	game.dialogue.Add(uiSystem)

	// Add systems in the order they should update
	game.gameplay.Add(system.NewAudioSystem())
	game.gameplay.Add(musicSystem)
	game.gameplay.Add(system.NewPlayerControllerSystem())
	game.gameplay.Add(system.NewPathfindingSystem())
	game.gameplay.Add(system.NewAINavigationSystem())
	game.gameplay.Add(system.NewAimSystem())
	game.gameplay.Add(animationSystem)
	game.gameplay.Add(system.NewColorSystem())
	game.gameplay.Add(system.NewWhiteFlashSystem())
	game.gameplay.Add(system.NewInvulnerabilitySystem())
	game.gameplay.Add(system.NewCombatSystem())
	game.gameplay.Add(system.NewDamageKnockbackSystem())
	game.gameplay.Add(system.NewArenaNodeSystem())
	game.gameplay.Add(system.NewPlayerHealthBarSystem())
	game.gameplay.Add(system.NewTrophyCounterSystem())
	game.gameplay.Add(system.NewHazardSystem())
	game.gameplay.Add(system.NewAnchorSystem())
	game.gameplay.Add(system.NewClusterRepulsionSystem())
	game.gameplay.Add(physicsSystem)
	game.gameplay.Add(dialogueInputSystem)
	game.gameplay.Add(dialoguePopupSystem)
	game.gameplay.Add(system.NewTriggerSystem())
	game.gameplay.Add(system.NewPickupHoverSystem())
	game.gameplay.Add(system.NewPickupCollectSystem())
	game.gameplay.Add(scriptSystem)
	game.gameplay.Add(system.NewTTLSystem())
	game.gameplay.Add(system.NewRespawnSystem())
	game.gameplay.Add(system.NewTransitionPopSystem())
	game.gameplay.Add(system.NewTransitionSystem())
	game.gameplay.Add(game.persistence)
	game.gameplay.Add(system.NewSpawnChildrenSystem())
	game.gameplay.Add(cameraSystem)
	game.gameplay.Add(system.NewParallaxSystem())

	game.camera = cameraSystem
	game.scriptRuntime = scriptSystem

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
		g.debugOverlay = !g.debugOverlay
	}

	if g.active == nil {
		g.active = g.gameplay
	}

	popupVisible := g.active == g.gameplay && g.dialoguePopupRequested()
	if g.active == g.gameplay && g.input != nil {
		g.input.Update(g.world)
		if popupVisible {
			g.clearAttackInputs()
		}
	}
	if g.active == g.dialogue {
		if g.pendingPress {
			g.setDialogueInputPressed(true)
			g.pendingPress = false
		} else if g.dialogueInput != nil {
			g.dialogueInput.Update(g.world)
		}
	}
	if g.active != nil {
		g.active.Update(g.world)
	}

	if g.active == g.gameplay && g.dialogueStartRequested() {
		g.active = g.dialogue
		g.dialogueOpen = false
		g.pendingPress = true
	} else if g.active == g.dialogue {
		if system.IsDialogueActive(g.world) {
			g.dialogueOpen = true
		} else if g.dialogueOpen {
			g.active = g.gameplay
			g.dialogueOpen = false
			g.setDialogueInputPressed(false)
		} else if !g.pendingPress {
			g.active = g.gameplay
			g.setDialogueInputPressed(false)
		}
	}

	if err := g.processPrefabEvents(); err != nil {
		panic("failed to process prefab events: " + err.Error())
	}

	return nil
}

func (g *Game) dialoguePopupRequested() bool {
	if g == nil || g.world == nil {
		return false
	}

	popupEntity, ok := ecs.First(g.world, component.DialoguePopupComponent.Kind())
	if !ok {
		return false
	}

	popup, ok := ecs.Get(g.world, popupEntity, component.DialoguePopupComponent.Kind())
	if !ok || popup == nil || popup.TargetDialogueEntity == 0 {
		return false
	}

	sprite, ok := ecs.Get(g.world, popupEntity, component.SpriteComponent.Kind())
	if !ok || sprite == nil {
		return false
	}

	return !sprite.Disabled
}

func (g *Game) dialogueStartRequested() bool {
	if !g.dialoguePopupRequested() {
		return false
	}

	inputEntity, ok := ecs.First(g.world, component.DialogueInputComponent.Kind())
	if !ok {
		return false
	}

	input, ok := ecs.Get(g.world, inputEntity, component.DialogueInputComponent.Kind())
	if !ok || input == nil {
		return false
	}

	return input.Pressed
}

func (g *Game) setDialogueInputPressed(pressed bool) {
	if g == nil || g.world == nil {
		return
	}

	inputEntity, ok := ecs.First(g.world, component.DialogueInputComponent.Kind())
	if !ok {
		return
	}

	input, ok := ecs.Get(g.world, inputEntity, component.DialogueInputComponent.Kind())
	if !ok || input == nil {
		return
	}

	input.Pressed = pressed
}

func (g *Game) clearAttackInputs() {
	if g == nil || g.world == nil {
		return
	}

	ecs.ForEach(g.world, component.InputComponent.Kind(), func(_ ecs.Entity, input *component.Input) {
		if input == nil {
			return
		}
		input.AttackPressed = false
		input.UpwardAttackPressed = false
	})
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.render != nil {
		g.render.Draw(g.world, screen)
	}
	if g.ui != nil {
		g.ui.Draw(g.world, screen)
	}

	if g.debugPhysics && g.physics != nil {
		system.DrawPhysicsDebug(g.physics.Space(), g.world, screen)
		system.DrawAIStateDebug(g.world, screen)
		system.DrawEntityIDDebug(g.world, screen)
		system.DrawPathfindingDebug(g.world, screen)
		system.DrawPickupDebug(g.world, screen)
		system.DrawTransitionDebug(g.world, screen)
		system.DrawHazardDebug(g.world, screen)
		system.DrawPlayerStateDebug(g.world, screen)
	}
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	if g.camera != nil {
		g.camera.SetScreenSize(common.BaseWidth, common.BaseHeight)
	}

	return common.BaseWidth, common.BaseHeight
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
